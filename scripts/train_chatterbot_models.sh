#!/usr/bin/env bash
set -euo pipefail

# Trains intent-nlu models from chatterbot-corpus language datasets.
# Default languages: zh,en
# Default outputs are saved inside repository:
# - datasets/generated/*.csv (effective training data)
# - datasets/generated/eval/*.json (training/evaluation metadata)
# - models/model-<lang> (trained model artifacts)
# - models/multilingual (merged multilingual bundle)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

LANGS="zh,en"
CORPUS_REPO_URL="https://github.com/gunthercox/chatterbot-corpus.git"
CORPUS_REPO_DIR="${REPO_ROOT}/.cache/chatterbot-corpus"
OUTPUT_DIR="${REPO_ROOT}/models"
DEFAULT_EXTRA_DIR="${REPO_ROOT}/datasets/default"
GENERATED_DATASET_DIR="${REPO_ROOT}/datasets/generated"
EVAL_REPORT_DIR="${REPO_ROOT}/datasets/generated/eval"
THRESHOLD="0.55"
UNKNOWN_INTENT="unknown"
INCLUDE_REPLIES="true"
SKIP_UNMAPPED="true"
SPLIT_ENABLED="true"
TRAIN_RATIO="0.8"
VAL_RATIO="0.1"
TEST_RATIO="0.1"
SEED="42"
AUTO_CALIBRATE="true"
DISABLE_TAXONOMY="true"
TAXONOMY_ALIASES=""
VERSION_PREFIX="$(date -u +%Y.%m.%d)"
MERGE_BUNDLE="true"
BUNDLE_DIR="${REPO_ROOT}/models/multilingual"
BUNDLE_VERSION=""
ROUTER_DEFAULT_LANG=""

CORPUS_COMMIT=""
CORPUS_HEAD=""

usage() {
  cat <<USAGE
Usage: $(basename "$0") [options]

Options:
  --langs <csv>                    Languages to train, default: zh,en
  --corpus-repo-url <url>          chatterbot-corpus git URL
  --corpus-repo-dir <dir>          Local corpus repo dir (cache)
  --output-dir <dir>               Output model root dir
  --default-extra-dir <dir>        Directory of default business CSV files (<lang>_business.csv)
  --generated-dataset-dir <dir>    Directory to dump effective training datasets
  --eval-report-dir <dir>          Directory to dump model metadata/evaluation reports
  --threshold <float>              Default threshold for each trained model
  --unknown-intent <name>          Unknown intent label
  --include-replies <bool>         Include all turns in conversations
  --skip-unmapped <bool>           Skip files without mapped intents
  --split-enabled <bool>           Enable deterministic train/val/test split
  --train-ratio <float>            Train split ratio
  --val-ratio <float>              Validation split ratio
  --test-ratio <float>             Test split ratio
  --seed <int>                     Split seed
  --auto-calibrate <bool>          Auto calibrate per-intent thresholds from validation split
  --disable-taxonomy <bool>        Disable built-in intent taxonomy normalization (default: true)
  --taxonomy-aliases <file>        Extra taxonomy alias YAML file
  --merge-bundle <bool>            Build merged multilingual bundle, default: true
  --bundle-dir <dir>               Output multilingual bundle dir
  --bundle-version <ver>           Bundle version (default auto generated)
  --router-default-lang <lang>     Default language in bundle manifest
  --version-prefix <prefix>        Model version prefix (suffix auto includes lang + timestamp)
  -h, --help                       Show this help

Examples:
  $(basename "$0")
  $(basename "$0") --langs zh,en,ja --output-dir ./models
USAGE
}

log() {
  printf '[intent-nlu-train] %s\n' "$*"
}

ensure_bool() {
  local value="${1:-}"
  if [[ "${value}" != "true" && "${value}" != "false" ]]; then
    echo "invalid boolean value: ${value}" >&2
    exit 1
  fi
}

language_to_corpus_dir() {
  local lang="$(echo "$1" | tr '[:upper:]' '[:lower:]')"
  case "${lang}" in
    zh|zh-cn|zh-hans|zh-tw|zh-hant|chinese) echo "chinese" ;;
    en|en-us|en-gb|english) echo "english" ;;
    ja|ja-jp|jp|japanese) echo "japanese" ;;
    ko|ko-kr|korean) echo "korean" ;;
    fr|fr-fr|french) echo "french" ;;
    de|de-de|german) echo "german" ;;
    es|es-es|spanish) echo "spanish" ;;
    it|it-it|italian) echo "italian" ;;
    pt|pt-br|pt-pt|portuguese) echo "portuguese" ;;
    ru|ru-ru|russian) echo "russian" ;;
    hi|hindi) echo "hindi" ;;
    no|norwegian) echo "norwegian" ;;
    sv|swedish) echo "swedish" ;;
    tr|turkish) echo "turkish" ;;
    *)
      # Allow direct folder name passthrough.
      echo "${lang}"
      ;;
  esac
}

normalize_lang_code() {
  local lang="$(echo "$1" | tr '[:upper:]' '[:lower:]')"
  case "${lang}" in
    zh-*|chinese) echo "zh" ;;
    en-*|english) echo "en" ;;
    ja-*|jp|japanese) echo "ja" ;;
    ko-*|korean) echo "ko" ;;
    *)
      if [[ "${#lang}" -ge 2 ]]; then
        echo "${lang:0:2}"
      else
        echo "${lang}"
      fi
      ;;
  esac
}

slugify() {
  local text="$1"
  text="$(echo "$text" | tr '[:upper:]' '[:lower:]')"
  text="$(echo "$text" | sed -E 's/[^a-z0-9]+/_/g; s/^_+//; s/_+$//')"
  if [[ -z "${text}" ]]; then
    text="general"
  fi
  echo "${text}"
}

generate_file_map() {
  local corpus_dir="$1"
  local map_file="$2"

  : > "${map_file}"
  local file_count=0

  while IFS= read -r -d '' file; do
    local base
    base="$(basename "${file}")"
    local stem="${base%.*}"
    local slug
    slug="$(slugify "${stem}")"
    printf '%s: chitchat_%s\n' "${base}" "${slug}" >> "${map_file}"
    file_count=$((file_count + 1))
  done < <(find "${corpus_dir}" -maxdepth 1 -type f \( -name '*.yml' -o -name '*.yaml' \) -print0 | sort -z)

  if [[ ${file_count} -eq 0 ]]; then
    echo "no yaml files found under ${corpus_dir}" >&2
    exit 1
  fi
}

prepare_corpus_repo() {
  mkdir -p "$(dirname "${CORPUS_REPO_DIR}")"
  if [[ -d "${CORPUS_REPO_DIR}/.git" ]]; then
    log "updating corpus repo: ${CORPUS_REPO_DIR}"
    git -C "${CORPUS_REPO_DIR}" fetch --depth 1 origin
    local head_ref
    head_ref="$(git -C "${CORPUS_REPO_DIR}" symbolic-ref refs/remotes/origin/HEAD 2>/dev/null || true)"
    if [[ -n "${head_ref}" ]]; then
      local branch="${head_ref#refs/remotes/origin/}"
      git -C "${CORPUS_REPO_DIR}" reset --hard "origin/${branch}"
    else
      git -C "${CORPUS_REPO_DIR}" reset --hard origin/main || git -C "${CORPUS_REPO_DIR}" reset --hard origin/master
    fi
  else
    log "cloning corpus repo to: ${CORPUS_REPO_DIR}"
    git clone --depth 1 "${CORPUS_REPO_URL}" "${CORPUS_REPO_DIR}"
  fi

  CORPUS_COMMIT="$(git -C "${CORPUS_REPO_DIR}" rev-parse HEAD)"
  CORPUS_HEAD="$(git -C "${CORPUS_REPO_DIR}" rev-parse --abbrev-ref HEAD || true)"
  log "corpus commit=${CORPUS_COMMIT} branch=${CORPUS_HEAD}"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --langs)
        LANGS="$2"; shift 2 ;;
      --corpus-repo-url)
        CORPUS_REPO_URL="$2"; shift 2 ;;
      --corpus-repo-dir)
        CORPUS_REPO_DIR="$2"; shift 2 ;;
      --output-dir)
        OUTPUT_DIR="$2"; shift 2 ;;
      --default-extra-dir)
        DEFAULT_EXTRA_DIR="$2"; shift 2 ;;
      --generated-dataset-dir)
        GENERATED_DATASET_DIR="$2"; shift 2 ;;
      --eval-report-dir)
        EVAL_REPORT_DIR="$2"; shift 2 ;;
      --threshold)
        THRESHOLD="$2"; shift 2 ;;
      --unknown-intent)
        UNKNOWN_INTENT="$2"; shift 2 ;;
      --include-replies)
        INCLUDE_REPLIES="$2"; shift 2 ;;
      --skip-unmapped)
        SKIP_UNMAPPED="$2"; shift 2 ;;
      --split-enabled)
        SPLIT_ENABLED="$2"; shift 2 ;;
      --train-ratio)
        TRAIN_RATIO="$2"; shift 2 ;;
      --val-ratio)
        VAL_RATIO="$2"; shift 2 ;;
      --test-ratio)
        TEST_RATIO="$2"; shift 2 ;;
      --seed)
        SEED="$2"; shift 2 ;;
      --auto-calibrate)
        AUTO_CALIBRATE="$2"; shift 2 ;;
      --disable-taxonomy)
        DISABLE_TAXONOMY="$2"; shift 2 ;;
      --taxonomy-aliases)
        TAXONOMY_ALIASES="$2"; shift 2 ;;
      --merge-bundle)
        MERGE_BUNDLE="$2"; shift 2 ;;
      --bundle-dir)
        BUNDLE_DIR="$2"; shift 2 ;;
      --bundle-version)
        BUNDLE_VERSION="$2"; shift 2 ;;
      --router-default-lang)
        ROUTER_DEFAULT_LANG="$2"; shift 2 ;;
      --version-prefix)
        VERSION_PREFIX="$2"; shift 2 ;;
      -h|--help)
        usage; exit 0 ;;
      *)
        echo "unknown option: $1" >&2
        usage
        exit 1 ;;
    esac
  done
}

declare -A TRAINED_MODEL_DIRS=()
declare -a TRAINED_LANGS=()

train_one_language() {
  local input_lang="$1"
  local corpus_subdir
  corpus_subdir="$(language_to_corpus_dir "${input_lang}")"
  local lang
  lang="$(normalize_lang_code "${input_lang}")"

  local corpus_dir="${CORPUS_REPO_DIR}/chatterbot_corpus/data/${corpus_subdir}"
  if [[ ! -d "${corpus_dir}" ]]; then
    echo "corpus dir not found for ${input_lang}: ${corpus_dir}" >&2
    exit 1
  fi

  mkdir -p "${GENERATED_DATASET_DIR}" "${OUTPUT_DIR}" "${EVAL_REPORT_DIR}"
  local map_file="${GENERATED_DATASET_DIR}/${lang}_file_map.yaml"
  local dump_samples_csv="${GENERATED_DATASET_DIR}/${lang}_train.csv"
  local eval_report="${EVAL_REPORT_DIR}/${lang}_meta.json"
  generate_file_map "${corpus_dir}" "${map_file}"

  local version="${VERSION_PREFIX}.${lang}.$(date -u +%H%M%S)"
  local model_dir="${OUTPUT_DIR}/model-${lang}"
  local extra_csv=""
  for csv_file in "${DEFAULT_EXTRA_DIR}/${lang}_business.csv" "${DEFAULT_EXTRA_DIR}/${lang}_skill_routing.csv"; do
    if [[ -f "${csv_file}" ]]; then
      [[ -n "${extra_csv}" ]] && extra_csv="${extra_csv},"
      extra_csv="${extra_csv}${csv_file}"
      log "using dataset: ${csv_file}"
    fi
  done

  log "training lang=${lang} corpus=${corpus_dir} out=${model_dir}"
  if [[ -z "${extra_csv}" ]]; then
    log "no default extra datasets for ${lang}, skip extra csv"
  fi

  (
    cd "${REPO_ROOT}"
    local cmd=(
      env GOWORK=off go run ./cmd/intent-nlu-train
      -lang "${lang}"
      -corpus-root "${corpus_dir}"
      -file-map "${map_file}"
      -skip-unmapped="${SKIP_UNMAPPED}"
      -include-replies="${INCLUDE_REPLIES}"
      -dump-samples "${dump_samples_csv}"
      -eval-report "${eval_report}"
      -out "${model_dir}"
      -version "${version}"
      -threshold "${THRESHOLD}"
      -unknown-intent "${UNKNOWN_INTENT}"
      -split-enabled="${SPLIT_ENABLED}"
      -train-ratio "${TRAIN_RATIO}"
      -val-ratio "${VAL_RATIO}"
      -test-ratio "${TEST_RATIO}"
      -seed "${SEED}"
      -auto-calibrate-thresholds="${AUTO_CALIBRATE}"
      -disable-taxonomy="${DISABLE_TAXONOMY}"
      -source-name "chatterbot-corpus"
      -source-version "${CORPUS_HEAD}"
      -source-revision "${corpus_subdir}"
      -source-repo-url "${CORPUS_REPO_URL}"
      -source-commit "${CORPUS_COMMIT}"
    )
    if [[ -n "${extra_csv}" ]]; then
      cmd+=( -extra-csv "${extra_csv}" )
    fi
    if [[ -n "${TAXONOMY_ALIASES}" ]]; then
      cmd+=( -taxonomy-aliases "${TAXONOMY_ALIASES}" )
    fi
    "${cmd[@]}"
  )

  TRAINED_MODEL_DIRS["${lang}"]="${model_dir}"
  TRAINED_LANGS+=("${lang}")
  log "trained lang=${lang} model=${model_dir} dataset=${dump_samples_csv} eval=${eval_report}"
}

assemble_multilingual_bundle() {
  if [[ "${MERGE_BUNDLE}" != "true" ]]; then
    return
  fi
  if [[ ${#TRAINED_LANGS[@]} -eq 0 ]]; then
    echo "no trained models for bundle" >&2
    exit 1
  fi

  local default_lang="${ROUTER_DEFAULT_LANG}"
  if [[ -z "${default_lang}" ]]; then
    default_lang="${TRAINED_LANGS[0]}"
  fi

  local bundle_version="${BUNDLE_VERSION}"
  if [[ -z "${bundle_version}" ]]; then
    bundle_version="${VERSION_PREFIX}.bundle.$(date -u +%H%M%S)"
  fi

  local models_spec=""
  local sorted_langs
  sorted_langs="$(printf '%s\n' "${TRAINED_LANGS[@]}" | sort)"
  models_spec=""
  local lang
  while IFS= read -r lang; do
    [[ -z "${lang}" ]] && continue
    if [[ -z "${models_spec}" ]]; then
      models_spec="${lang}=${TRAINED_MODEL_DIRS[${lang}]}"
    else
      models_spec="${models_spec},${lang}=${TRAINED_MODEL_DIRS[${lang}]}"
    fi
  done <<< "${sorted_langs}"

  local training_params
  training_params="threshold=${THRESHOLD},unknown_intent=${UNKNOWN_INTENT},split_enabled=${SPLIT_ENABLED},train_ratio=${TRAIN_RATIO},val_ratio=${VAL_RATIO},test_ratio=${TEST_RATIO},seed=${SEED},auto_calibrate=${AUTO_CALIBRATE},disable_taxonomy=${DISABLE_TAXONOMY},include_replies=${INCLUDE_REPLIES},skip_unmapped=${SKIP_UNMAPPED}"

  (
    cd "${REPO_ROOT}"
    GOWORK=off go run ./cmd/intent-nlu-bundle \
      -bundle-dir "${BUNDLE_DIR}" \
      -models "${models_spec}" \
      -default-lang "${default_lang}" \
      -version "${bundle_version}" \
      -corpus-repo-url "${CORPUS_REPO_URL}" \
      -corpus-commit "${CORPUS_COMMIT}" \
      -training-params "${training_params}"
  )

  log "multilingual bundle generated: ${BUNDLE_DIR}"
}

main() {
  parse_args "$@"
  ensure_bool "${INCLUDE_REPLIES}"
  ensure_bool "${SKIP_UNMAPPED}"
  ensure_bool "${SPLIT_ENABLED}"
  ensure_bool "${AUTO_CALIBRATE}"
  ensure_bool "${DISABLE_TAXONOMY}"
  ensure_bool "${MERGE_BUNDLE}"

  if [[ -n "${TAXONOMY_ALIASES}" && ! -f "${TAXONOMY_ALIASES}" ]]; then
    echo "taxonomy alias file not found: ${TAXONOMY_ALIASES}" >&2
    exit 1
  fi

  prepare_corpus_repo

  IFS=',' read -r -a languages <<< "${LANGS}"
  if [[ ${#languages[@]} -eq 0 ]]; then
    echo "no languages provided" >&2
    exit 1
  fi

  local raw
  for raw in "${languages[@]}"; do
    local local_lang
    local_lang="$(echo "${raw}" | xargs)"
    if [[ -z "${local_lang}" ]]; then
      continue
    fi
    train_one_language "${local_lang}"
  done

  assemble_multilingual_bundle
  log "all done. models are under ${OUTPUT_DIR}; datasets under ${GENERATED_DATASET_DIR}; eval reports under ${EVAL_REPORT_DIR}"
}

main "$@"
