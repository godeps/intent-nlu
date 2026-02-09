#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

INPUT=""
DEFAULT_DIR="${REPO_ROOT}/datasets/default"
REVIEW_DIR="${REPO_ROOT}/datasets/feedback/review"
LOW_CONFIDENCE="0.6"
UNKNOWN_INTENT="unknown"

usage() {
  cat <<USAGE
Usage: $(basename "$0") --input <feedback.csv> [options]

Options:
  --input <file>              Feedback CSV path (required)
  --default-dir <dir>         Business dataset directory
  --review-dir <dir>          Review queue directory
  --low-confidence <float>    Threshold for low-confidence review
  --unknown-intent <name>     Unknown intent label
  -h, --help                  Show this help

Input CSV supported columns:
  text, language, pred_intent|intent, confidence|score, final_intent|human_intent|label
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input)
      INPUT="$2"; shift 2 ;;
    --default-dir)
      DEFAULT_DIR="$2"; shift 2 ;;
    --review-dir)
      REVIEW_DIR="$2"; shift 2 ;;
    --low-confidence)
      LOW_CONFIDENCE="$2"; shift 2 ;;
    --unknown-intent)
      UNKNOWN_INTENT="$2"; shift 2 ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      echo "unknown option: $1" >&2
      usage
      exit 1 ;;
  esac

done

if [[ -z "${INPUT}" ]]; then
  echo "--input is required" >&2
  usage
  exit 1
fi

cd "${REPO_ROOT}"
GOWORK=off go run ./cmd/chat-nlu-feedback \
  -input "${INPUT}" \
  -default-dir "${DEFAULT_DIR}" \
  -review-dir "${REVIEW_DIR}" \
  -low-confidence "${LOW_CONFIDENCE}" \
  -unknown-intent "${UNKNOWN_INTENT}"
