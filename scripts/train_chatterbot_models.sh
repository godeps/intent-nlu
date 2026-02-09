#!/usr/bin/env bash
set -euo pipefail

# Trains chat-nlu models from chatterbot-corpus language datasets.
# Default languages: zh,en
# Default outputs are saved inside repository:
# - datasets/generated/*.csv (effective training data)
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
THRESHOLD="0.55"
UNKNOWN_INTENT="unknown"
INCLUDE_REPLIES="true"
SKIP_UNMAPPED="true"
VERSION_PREFIX="$(date -u +%Y.%m.%d)"
MERGE_BUNDLE="true"
BUNDLE_DIR="${REPO_ROOT}/models/multilingual"
ROUTER_DEFAULT_LANG=""

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
  --threshold <float>              Default threshold for each trained model
  --unknown-intent <name>          Unknown intent label
  --include-replies <bool>         Include all turns in conversations
  --skip-unmapped <bool>           Skip files without mapped intents
  --merge-bundle <bool>            Build merged multilingual bundle, default: true
  --bundle-dir <dir>               Output multilingual bundle dir
  --router-default-lang <lang>     Default language in bundle manifest
  --version-prefix <prefix>        Model version prefix (suffix auto includes lang + timestamp)
  -h, --help                       Show this help

Examples:
  $(basename "$0")
  $(basename "$0") --langs zh,en,ja --output-dir ./models
USAGE
}

log() {
  printf '[chat-nlu-train] %s\n' "$*"
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
    git -C "${CORPUS_REPO_DIR}" reset --hard origin/master 2>/dev/null || \
      git -C "${CORPUS_REPO_DIR}" reset --hard origin/main
  else
    log "cloning corpus repo to: ${CORPUS_REPO_DIR}"
    git clone --depth 1 "${CORPUS_REPO_URL}" "${CORPUS_REPO_DIR}"
  fi
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
      --threshold)
        THRESHOLD="$2"; shift 2 ;;
      --unknown-intent)
        UNKNOWN_INTENT="$2"; shift 2 ;;
      --include-replies)
        INCLUDE_REPLIES="$2"; shift 2 ;;
      --skip-unmapped)
        SKIP_UNMAPPED="$2"; shift 2 ;;
      --merge-bundle)
        MERGE_BUNDLE="$2"; shift 2 ;;
      --bundle-dir)
        BUNDLE_DIR="$2"; shift 2 ;;
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

  mkdir -p "${GENERATED_DATASET_DIR}" "${OUTPUT_DIR}"
  local map_file="${GENERATED_DATASET_DIR}/${lang}_file_map.yaml"
  local dump_samples_csv="${GENERATED_DATASET_DIR}/${lang}_train.csv"
  generate_file_map "${corpus_dir}" "${map_file}"

  local version="${VERSION_PREFIX}.${lang}.$(date -u +%H%M%S)"
  local model_dir="${OUTPUT_DIR}/model-${lang}"
  local extra_csv="${DEFAULT_EXTRA_DIR}/${lang}_business.csv"

  log "training lang=${lang} corpus=${corpus_dir} out=${model_dir}"
  if [[ -f "${extra_csv}" ]]; then
    log "using default extra dataset: ${extra_csv}"
  else
    extra_csv=""
    log "no default extra dataset for ${lang}, skip extra csv"
  fi

  (
    cd "${REPO_ROOT}"
    if [[ -n "${extra_csv}" ]]; then
      GOWORK=off go run ./cmd/chat-nlu-train \
        -lang "${lang}" \
        -corpus-root "${corpus_dir}" \
        -file-map "${map_file}" \
        -skip-unmapped="${SKIP_UNMAPPED}" \
        -include-replies="${INCLUDE_REPLIES}" \
        -extra-csv "${extra_csv}" \
        -dump-samples "${dump_samples_csv}" \
        -out "${model_dir}" \
        -version "${version}" \
        -threshold "${THRESHOLD}" \
        -unknown-intent "${UNKNOWN_INTENT}"
    else
      GOWORK=off go run ./cmd/chat-nlu-train \
        -lang "${lang}" \
        -corpus-root "${corpus_dir}" \
        -file-map "${map_file}" \
        -skip-unmapped="${SKIP_UNMAPPED}" \
        -include-replies="${INCLUDE_REPLIES}" \
        -dump-samples "${dump_samples_csv}" \
        -out "${model_dir}" \
        -version "${version}" \
        -threshold "${THRESHOLD}" \
        -unknown-intent "${UNKNOWN_INTENT}"
    fi
  )

  TRAINED_MODEL_DIRS["${lang}"]="${model_dir}"
  TRAINED_LANGS+=("${lang}")
  log "trained lang=${lang} model=${model_dir} dataset=${dump_samples_csv}"
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

  local bundle_models_dir="${BUNDLE_DIR}/models"
  rm -rf "${BUNDLE_DIR}"
  mkdir -p "${bundle_models_dir}"

  local lang
  for lang in "${TRAINED_LANGS[@]}"; do
    local src_dir="${TRAINED_MODEL_DIRS[${lang}]}"
    local dst_dir="${bundle_models_dir}/${lang}"
    mkdir -p "${dst_dir}"
    cp "${src_dir}/model.gob" "${dst_dir}/model.gob"
    cp "${src_dir}/meta.json" "${dst_dir}/meta.json"
  done

  local manifest_path="${BUNDLE_DIR}/manifest.json"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local version="${VERSION_PREFIX}.bundle.$(date -u +%H%M%S)"

  {
    echo '{'
    echo "  \"version\": \"${version}\"," 
    echo "  \"createdAt\": \"${now}\"," 
    echo "  \"defaultLanguage\": \"${default_lang}\"," 
    echo '  "models": {'
    local i=0
    local total=${#TRAINED_LANGS[@]}
    for lang in "${TRAINED_LANGS[@]}"; do
      i=$((i + 1))
      local comma=","
      if [[ ${i} -eq ${total} ]]; then
        comma=""
      fi
      echo "    \"${lang}\": \"models/${lang}\"${comma}"
    done
    echo '  }'
    echo '}'
  } > "${manifest_path}"

  log "multilingual bundle generated: ${BUNDLE_DIR}"
}

main() {
  parse_args "$@"
  ensure_bool "${INCLUDE_REPLIES}"
  ensure_bool "${SKIP_UNMAPPED}"
  ensure_bool "${MERGE_BUNDLE}"

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
  log "all done. models are under ${OUTPUT_DIR}; datasets under ${GENERATED_DATASET_DIR}"
}

main "$@"
