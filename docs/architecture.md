# chat-nlu Architecture

## Overview

`chat-nlu` is a lightweight intent recognition module:

- Chinese tokenization: `gse`
- Other language tokenization: lightweight normalized token splitter
- Classification: `jbrukh/bayesian`
- Inference policy: confidence thresholds + unknown fallback

## Components

- `tokenizer.go`: normalization, punctuation stripping, stopword filtering, language-aware tokenization.
- `trainer.go`: supervised training from labeled samples.
- `model.go`: persist/load artifacts (`model.gob`, `meta.json`).
- `engine.go`: concurrent-safe single-model prediction and hot reload.
- `router.go`: multi-language model routing by hint or auto detection.
- `router_bundle.go`: multilingual bundle manifest and bundle loader.
- `language.go`: language normalization and lightweight language detection.
- `dataset/chatterbot/loader.go`: load chatterbot corpus (`*.yml`) into labeled samples.
- `cmd/chat-nlu-train`: train model from chatterbot and/or extra CSV.
- `cmd/chat-nlu-predict`: run prediction from model artifacts (single or multi-model).
- `scripts/train_chatterbot_models.sh`: one-click corpus download + zh/en training + bundle generation.

## Inference Flow

Single model (`Engine`):

1. Normalize input text.
2. Tokenize by model language.
3. Get class probabilities from bayesian model.
4. Apply threshold (per intent or default).
5. Return matched intent or unknown.

Multi model (`Router`):

1. Resolve language by `LanguageHint` or `DetectLanguage`.
2. Route to corresponding `Engine`.
3. Run single-model inference.

Bundle model:

1. `scripts/train_chatterbot_models.sh` produces `models/multilingual/manifest.json`.
2. `NewRouterFromBundle()` loads all language models from bundle paths.

## Model Files

- `model.gob`: classifier binary.
- `meta.json`: language, classes, thresholds, tokenizer settings, version.

## Integration Pattern

Use in pre-LLM stage:

1. Predict intent from user text.
2. If matched and high confidence, route deterministic tools.
3. Otherwise, fallback to LLM.
