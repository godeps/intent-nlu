# chat-nlu

A lightweight, reusable intent recognition toolkit for Go projects.

- Tokenization:
`github.com/go-ego/gse` for Chinese, lightweight tokenizer for other languages.
- Classification:
`github.com/jbrukh/bayesian`.
- Module:
`github.com/godeps/chat-nlu`

Chinese documentation:
`README.zh-CN.md`

## What It Solves

1. Fast pre-LLM intent routing.
2. Multi-language model routing (`zh`, `en`, more optional).
3. One-click corpus-based model training from `chatterbot-corpus`.
4. Reusable model bundle format for deployment.

## Repository Structure

```text
chat-nlu/
  cmd/
    chat-nlu-train/            # CLI: train one language model
    chat-nlu-predict/          # CLI: predict from one model / multi-model / bundle
  dataset/
    chatterbot/                # chatterbot corpus loader
  datasets/
    default/                   # default business CSV data, committed
      zh_business.csv
      en_business.csv
    generated/                 # generated effective train datasets
      *.csv
      *_file_map.yaml
  docs/
    architecture.md
  examples/
    file_intent_map.yaml
  models/
    model-zh/                  # trained zh model artifacts
      model.gob
      meta.json
    model-en/                  # trained en model artifacts
      model.gob
      meta.json
    multilingual/              # merged bundle
      manifest.json
      models/
        zh/
          model.gob
          meta.json
        en/
          model.gob
          meta.json
  scripts/
    train_chatterbot_models.sh # one-click corpus download + training + bundle

  # core package files
  types.go
  language.go
  tokenizer.go
  trainer.go
  model.go
  engine.go
  router.go
  router_bundle.go
```

## Core Concepts

1. Single model:
one language model directory containing `model.gob` and `meta.json`.
2. Multi-model routing:
map language to model directory (for example `zh=...`, `en=...`).
3. Bundle:
a merged directory with `manifest.json` + per-language model folders.

## Install

```bash
go get github.com/godeps/chat-nlu
```

## Quick Start

### 1) One-click training (default `zh,en`)

```bash
cd ../godeps/chat-nlu
./scripts/train_chatterbot_models.sh
```

Default outputs:

1. Models:
`./models/model-zh`, `./models/model-en`
2. Bundle:
`./models/multilingual`
3. Effective train datasets:
`./datasets/generated/zh_train.csv`, `./datasets/generated/en_train.csv`

### 2) Predict from bundle

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -bundle ./models/multilingual \
  -text "明天星期几" \
  -lang auto
```

### 3) Predict from single model

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -model ./models/model-zh \
  -text "你好" \
  -lang auto
```

## Training Methods

## Method A: one-click corpus training script

Script:
`./scripts/train_chatterbot_models.sh`

What it does:

1. Clone/update `https://github.com/gunthercox/chatterbot-corpus` to local cache.
2. Load corpus by language folder.
3. Auto-generate file-to-intent mapping as `chitchat_<filename>`.
4. Auto-append default business CSV from `datasets/default/<lang>_business.csv` if exists.
5. Train language models.
6. Dump effective training data to `datasets/generated`.
7. Build multilingual bundle into `models/multilingual`.

Common options:

```bash
./scripts/train_chatterbot_models.sh \
  --langs zh,en,ja \
  --threshold 0.5 \
  --output-dir ./models \
  --generated-dataset-dir ./datasets/generated \
  --merge-bundle true \
  --bundle-dir ./models/multilingual \
  --router-default-lang zh
```

Important options:

1. `--langs`:
comma-separated languages, default `zh,en`.
2. `--threshold`:
default confidence threshold for trained models.
3. `--default-extra-dir`:
directory for default business csv (`<lang>_business.csv`).
4. `--generated-dataset-dir`:
where effective training datasets are dumped.
5. `--merge-bundle`:
whether to generate merged bundle.
6. `--router-default-lang`:
default routing language in bundle manifest.

## Method B: manual CLI training

Single language from chatterbot + extra csv:

```bash
GOWORK=off go run ./cmd/chat-nlu-train \
  -lang zh \
  -corpus-root /path/to/chatterbot_corpus/data/chinese \
  -file-map ./examples/file_intent_map.yaml \
  -extra-csv ./datasets/default/zh_business.csv \
  -dump-samples ./datasets/generated/zh_train.csv \
  -out ./models/model-zh \
  -version 2026.02.09.zh.1 \
  -threshold 0.55
```

Training flags summary:

1. `-lang`:
model language (`zh/en/ja/ko/...`).
2. `-corpus-root`:
chatterbot language folder path.
3. `-file-map`, `-category-map`:
corpus mapping YAML files.
4. `-default-intent`:
default intent for unmapped files.
5. `-skip-unmapped`, `-include-replies`:
corpus loading strategy.
6. `-extra-csv`:
custom labeled csv with `text,intent`.
7. `-dump-samples`:
save effective train dataset as csv.
8. `-threshold`, `-thresholds`:
confidence thresholds.
9. `-out`, `-version`:
output model directory and version.

## Prediction Methods

## Method A: single model

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -model ./models/model-zh \
  -text "明天星期几" \
  -lang auto \
  -topk 3
```

## Method B: multi-model mapping

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -models "zh=./models/model-zh,en=./models/model-en" \
  -text "what is the weather tomorrow" \
  -lang auto
```

## Method C: bundle

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -bundle ./models/multilingual \
  -text "hello" \
  -lang auto
```

Predict flags summary:

1. `-model`:
single model dir.
2. `-models`:
comma-separated `lang=path` map.
3. `-bundle`:
bundle dir with `manifest.json`.
4. `-lang`:
language hint (`auto/zh/en/...`).
5. `-min-confidence`:
override threshold per request.
6. `-topk`:
number of candidates returned.

## Package Usage

Single model:

```go
engine, err := chatnlu.NewEngineFromDir("./models/model-zh")
if err != nil {
    panic(err)
}

pred, err := engine.Predict(context.Background(), "明天星期几", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "auto",
})
```

Router from model map:

```go
router, err := chatnlu.NewRouterFromDirs(map[string]string{
    "zh": "./models/model-zh",
    "en": "./models/model-en",
}, "zh")
if err != nil {
    panic(err)
}

pred, _ := router.Predict(context.Background(), "what is weather tomorrow", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "auto",
})
```

Router from bundle:

```go
router, err := chatnlu.NewRouterFromBundle("./models/multilingual")
if err != nil {
    panic(err)
}

pred, _ := router.Predict(context.Background(), "你好", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "auto",
})
```

## Notes And Caveats

1. `chatterbot-corpus` is mostly chitchat data.
You should add business datasets for real intents like `calendar_info` and `weather_info`.
2. Threshold tuning matters.
High threshold improves precision but may increase `unknown`.
3. Bundle is a deployment package, not one merged classifier.
Each language still uses its own model.
4. Keep generated data and model artifacts versioned.
Reproducibility depends on corpus revision + mapping + threshold.
5. Large generated files can grow repository size.
If needed, move artifacts to release storage and keep only manifest snapshots in git.
6. The local cache directory is `./.cache`.
It is ignored by git and can be safely rebuilt.

## Troubleshooting

1. YAML parse errors in corpus files:
update to latest loader and rerun script.
2. Predictions always `unknown`:
reduce threshold with `-min-confidence` for testing and re-tune training threshold.
3. Wrong language routing:
set explicit `-lang` hint or router default language.
4. Missing expected intent labels:
verify `file_map` and extra business CSV label quality.

## Testing

```bash
cd ../godeps/chat-nlu
GOWORK=off go test ./...
```
