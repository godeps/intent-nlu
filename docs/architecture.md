# chat-nlu Architecture

## Overview

`chat-nlu` is a lightweight intent recognition package focused on low-latency pre-LLM routing and reusable deployment.

Core stack:

- Chinese tokenization: `gse`
- Non-CJK tokenization: normalized splitter
- Classifier: `jbrukh/bayesian`

## Main Components

- `tokenizer.go`: language-aware tokenization and normalization.
- `language.go`: language normalization and lightweight language detection.
- `taxonomy.go`: intent alias normalization (taxonomy).
- `evaluation.go`: deterministic split, threshold calibration, metrics and confusion matrix.
- `trainer.go`: supervised training and metadata generation.
- `model.go`: model persistence and loading (`model.gob`, `meta.json`).
- `engine.go`: single-model prediction and hot reload.
- `router.go`: multi-language routing with optional cross-language raw-score fallback.
- `router_bundle.go`: bundle manifest and bundle loader.
- `embedded_bundle.go`: embedded bundle extraction and router loading for dependency consumers.
- `hybrid_policy.go`: deterministic rules + NLU + fallback orchestration.
- `dataset/chatterbot/loader.go`: chatterbot corpus loader.

## Runtime Flow

### 1) Pre-LLM phase

1. Optional deterministic rule matching.
2. NLU prediction (`Engine` or `Router`).
3. If matched with confidence, return deterministic route/tool intent.
4. If unknown/low confidence, go fallback path (typically call LLM).

### 2) Fallback/LLM phase

`chat-nlu` does not replace LLM generation; it decides when to skip/route/escalate.

### 3) Feedback phase

Prediction logs + human labels are fed into `cmd/chat-nlu-feedback`:

- high-quality labels -> business dataset (`datasets/default/*_business.csv`)
- low-confidence/unknown -> review queue (`datasets/feedback/review`)

## Training & Evaluation Flow

1. Load data from chatterbot and/or business CSV.
2. Optionally apply taxonomy normalization (disabled by default).
3. Tokenize samples.
4. Deterministically split into train/val/test (hash + seed).
5. Train bayesian classifier on train split.
6. Optional threshold calibration on validation split.
7. Evaluate train/val/test with:
   - accuracy
   - macro/micro F1
   - per-intent precision/recall/F1
   - confusion matrix
8. Save model and metadata.

## Artifacts

### Single model directory

- `model.gob`
- `meta.json`

`meta.json` includes:

- version/language/threshold policy
- classes + canonical intents + aliases
- tokenizer config
- training metadata (split sizes, seed, calibration)
- evaluation report by split
- source metadata

### Bundle directory

- `manifest.json`
- `models/<lang>/{model.gob,meta.json}`

`manifest.json` includes:

- model map (`lang -> relative path`)
- default language
- corpus metadata
- training parameter snapshot
- per-model summary

## Design Tradeoffs

1. Prioritizes latency and explainability over deep semantic modeling.
2. Bayesian model is easy to train/deploy, but needs quality labels for domain intents.
3. Cross-language routing fallback improves robustness for short/mixed input, with small extra compute.
4. Taxonomy normalization can reduce intent drift, but default is disabled to preserve fine-grained classes.

## Integration Pattern

Recommended production chain:

1. `HybridPolicy` (rule first)
2. NLU inference (`Router`)
3. If `fallback`, call LLM/tool planner
4. Capture feedback and retrain periodically
