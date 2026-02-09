# chat-nlu

Lightweight, reusable intent recognition toolkit for Go projects.

- Module: `github.com/godeps/chat-nlu`
- Chinese docs: `README.zh-CN.md`
- Core stack:
  - Tokenization: `github.com/go-ego/gse` (Chinese), normalized splitter for non-CJK
  - Classifier: `github.com/jbrukh/bayesian`

## What It Provides

1. Low-latency pre-LLM intent recognition.
2. Deterministic train/val/test evaluation pipeline.
3. Per-intent threshold calibration (optional).
4. Intent taxonomy normalization (aliases -> canonical intents).
5. Multi-language routing (`zh`, `en`, extensible).
6. Hybrid policy (`rules -> NLU -> fallback/LLM`).
7. Data feedback loop for active dataset improvement.
8. Reproducibility metadata in model meta and bundle manifest.

## Repository Layout

```text
chat-nlu/
  cmd/
    chat-nlu-train/            # train one language model
    chat-nlu-predict/          # predict by single model / model map / bundle
    chat-nlu-bundle/           # build multilingual bundle from trained models
    chat-nlu-feedback/         # feedback ingestion and dataset/review update
  dataset/chatterbot/          # chatterbot corpus loader
  datasets/
    default/
      zh_business.csv
      en_business.csv
    generated/
      *_train.csv              # effective training samples
      *_file_map.yaml          # auto-generated chatterbot mappings
      eval/*.json              # evaluation/model meta snapshots
    feedback/
      review/                  # low-confidence/unknown queue
      archive/                 # optional archived feedback
  docs/
    architecture.md
  examples/
    file_intent_map.yaml
  models/
    model-zh/
    model-en/
    multilingual/
  scripts/
    train_chatterbot_models.sh
    feedback_loop.sh

  # core package
  types.go
  tokenizer.go
  language.go
  taxonomy.go
  evaluation.go
  trainer.go
  model.go
  engine.go
  router.go
  router_bundle.go
  hybrid_policy.go
```

## Quick Start

### 1) Run tests

```bash
cd ../godeps/chat-nlu
make test
```

### 2) Reproducible local evaluation (`make eval`)

```bash
cd ../godeps/chat-nlu
make eval
```

Outputs:

- `datasets/generated/eval/zh_eval.json`
- `datasets/generated/eval/en_eval.json`

Both reports include split metrics (accuracy, macro-F1, micro-F1, confusion matrix, per-intent metrics) and training metadata.

### 3) One-click corpus training (`zh,en` default)

```bash
cd ../godeps/chat-nlu
./scripts/train_chatterbot_models.sh
```

Outputs:

1. `models/model-zh`, `models/model-en`
2. `models/multilingual`
3. `datasets/generated/{zh,en}_train.csv`
4. `datasets/generated/eval/{zh,en}_meta.json`

## Default Matrix (Tables)

Default embedded models currently use taxonomy disabled and fine-grained intents:

- `zh`: 20 classes
- `en`: 24 classes
- fallback: `unknown`

### Current Intent Classes

| Intent | Description | zh | en |
| --- | --- | --- | --- |
| `calendar_info` | Date, weekday, holiday, lunar/solar calendar style queries | Yes | Yes |
| `weather_info` | Weather forecast, rain, temperature, climate-style questions | Yes | Yes |
| `chitchat_greeting` | Direct short greetings (hi/hello/good morning) | Yes | Yes |
| `chitchat_greetings` | Greeting variants from chatterbot corpus | Yes | Yes |
| `chitchat_ai` | AI/assistant topic small talk | Yes | Yes |
| `chitchat_botprofile` | Bot identity/capabilities/preferences | Yes | Yes |
| `chitchat_conversations` | Generic open-domain conversation | Yes | Yes |
| `chitchat_emotion` | Emotion/mood/support style casual talk | Yes | Yes |
| `chitchat_food` | Food and drink discussion | Yes | Yes |
| `chitchat_gossip` | Gossip/celebrity/light social topics | Yes | Yes |
| `chitchat_history` | History trivia chat | Yes | Yes |
| `chitchat_humor` | Jokes/funny content | Yes | Yes |
| `chitchat_literature` | Books/writing/literature chat | Yes | Yes |
| `chitchat_money` | Money/finance-light casual talk | Yes | Yes |
| `chitchat_movies` | Movies/entertainment chat | Yes | Yes |
| `chitchat_politics` | Politics social chat | Yes | Yes |
| `chitchat_psychology` | Psychology/personality casual topics | Yes | Yes |
| `chitchat_science` | Science trivia chat | Yes | Yes |
| `chitchat_sports` | Sports chat | Yes | Yes |
| `chitchat_trivia` | General trivia/knowledge snippets | Yes | Yes |
| `chitchat_coding` | Programming/dev casual topics | No | Yes |
| `chitchat_computers` | Computer/device/software casual topics | No | Yes |
| `chitchat_health` | Health/wellness casual topics | No | Yes |
| `chitchat_tech_support` | Light technical support style chat | No | Yes |

### Supported Languages

| Language | Code | Default Embedded Model | Auto Detect | Notes |
| --- | --- | --- | --- | --- |
| Chinese | `zh` | Yes | Yes | `gse` tokenizer, default class count 20 |
| English | `en` | Yes | Yes | normalized tokenizer, default class count 24 |
| Japanese | `ja` | No (train yourself) | Yes | language detection supported, no default model shipped |
| Korean | `ko` | No (train yourself) | Yes | language detection supported, no default model shipped |

### Recommended Confidence (Trust Threshold)

| Use Case | Suggested Threshold | Behavior |
| --- | --- | --- |
| Strict tool routing (high precision) | `0.75 - 0.85` | Only high-confidence intents are accepted; fallback increases |
| Business routing (balanced) | `0.60 - 0.70` | Good default for production intent dispatch |
| Default baseline | `0.55` | Current training default if no per-intent override |
| Recall-first exploration | `0.40 - 0.55` | More matched intents, but more false positives |
| Uncertain intent | `< threshold` => `unknown` | Route to fallback or LLM |

### Operational Notes

| Topic | Risk | Recommendation |
| --- | --- | --- |
| Fine-grained classes | Confusion across similar chitchat intents | Keep enough per-intent samples and evaluate confusion matrix |
| Corpus bias | chatterbot data is mostly chitchat | Always mix business CSV for production tasks |
| Multilingual routing | Short/mixed text can route wrong language | Use language hint for critical paths |
| Threshold drift | Retraining changes confidence distribution | Re-calibrate thresholds and compare `eval/*.json` every release |
| Embedded bundle updates | New models require dependency rebuild | Pin model version and release notes with each update |

## Training Workflows

### A) One-click script (recommended)

```bash
./scripts/train_chatterbot_models.sh \
  --langs zh,en \
  --threshold 0.55 \
  --split-enabled true \
  --train-ratio 0.8 \
  --val-ratio 0.1 \
  --test-ratio 0.1 \
  --seed 42 \
  --auto-calibrate true \
  --merge-bundle true \
  --bundle-dir ./models/multilingual
```

What it does:

1. Clone/update `chatterbot-corpus`.
2. Auto-generate file->intent mapping (`chitchat_<file>`).
3. Merge optional business CSV (`datasets/default/<lang>_business.csv`).
4. Train models with split/evaluation/calibration.
5. Build multilingual bundle via `cmd/chat-nlu-bundle`.

### B) Manual training CLI

```bash
GOWORK=off go run ./cmd/chat-nlu-train \
  -lang zh \
  -corpus-root /path/to/chatterbot_corpus/data/chinese \
  -file-map ./examples/file_intent_map.yaml \
  -extra-csv ./datasets/default/zh_business.csv \
  -dump-samples ./datasets/generated/zh_train.csv \
  -eval-report ./datasets/generated/eval/zh_meta.json \
  -out ./models/model-zh \
  -version 2026.02.09.zh.1 \
  -threshold 0.55 \
  -split-enabled=true \
  -train-ratio 0.8 \
  -val-ratio 0.1 \
  -test-ratio 0.1 \
  -seed 42 \
  -auto-calibrate-thresholds=true
```

Important flags:

- Data: `-corpus-root`, `-file-map`, `-category-map`, `-extra-csv`
- Split/eval: `-split-enabled`, `-train-ratio`, `-val-ratio`, `-test-ratio`, `-seed`, `-eval-report`
- Threshold: `-threshold`, `-thresholds`, `-auto-calibrate-thresholds`
- Taxonomy: `-disable-taxonomy` (default `true`), `-taxonomy-aliases`
- Reproducibility source: `-source-name`, `-source-version`, `-source-revision`, `-source-repo-url`, `-source-commit`

## Bundle Build CLI

```bash
GOWORK=off go run ./cmd/chat-nlu-bundle \
  -bundle-dir ./models/multilingual \
  -models "zh=./models/model-zh,en=./models/model-en" \
  -default-lang zh \
  -version 2026.02.09.bundle.1 \
  -corpus-repo-url https://github.com/gunthercox/chatterbot-corpus.git \
  -corpus-commit <commit> \
  -training-params "seed=42,train_ratio=0.8,val_ratio=0.1,test_ratio=0.1"
```

Bundle manifest includes:

- model mapping (`lang -> relative path`)
- corpus metadata
- training parameter snapshot
- per-model summary

## Prediction

### Single model

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -model ./models/model-zh \
  -text "明天星期几" \
  -lang auto \
  -topk 3
```

### Multi-model map

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -models "zh=./models/model-zh,en=./models/model-en" \
  -text "weather tomorrow" \
  -lang auto
```

### Bundle

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -bundle ./models/multilingual \
  -text "你好" \
  -lang auto
```

### No model flags (use embedded default bundle)

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -text "hello" \
  -lang auto
```

If `-bundle`, `-models`, and `-model` are all omitted, the command loads embedded default models.

## Embedded Bundle (for dependency consumers)

`chat-nlu` can embed default multilingual models into the package itself.
When another Go service imports this module, it can load models without shipping external files.

```go
router, err := chatnlu.NewRouterFromEmbedded()
if err != nil {
    panic(err)
}

pred, err := router.Predict(context.Background(), "hello", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "en",
})
```

Optional custom extraction cache directory:

```go
router, err := chatnlu.NewRouterFromEmbeddedIn("./.cache/chat-nlu")
```

## Feedback Loop

Use model feedback CSV to:

1. Append human-labeled rows into `datasets/default/<lang>_business.csv`
2. Put low-confidence/unknown rows into review queue

```bash
./scripts/feedback_loop.sh --input ./tmp/feedback.csv
```

Supported CSV headers:

- required: `text`
- optional aliases:
  - language: `language`
  - predicted intent: `pred_intent|intent|predicted_intent`
  - score: `confidence|score`
  - human label: `final_intent|human_intent|label`

## Package Usage

### Engine

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

### Router

```go
router, err := chatnlu.NewRouterFromBundle("./models/multilingual")
if err != nil {
    panic(err)
}

pred, err := router.Predict(context.Background(), "weather tomorrow", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "auto",
})
```

### Hybrid Policy (rules -> NLU -> fallback)

```go
policy := &chatnlu.HybridPolicy{
    Router: router,
    Rules: []chatnlu.DeterministicRule{
        {ID: "r1", Intent: "calendar_info", ContainsAny: []string{"星期几", "几号"}},
    },
}
_ = policy.Prepare()

decision, err := policy.Decide(context.Background(), userText, chatnlu.PredictOptions{TopK: 3})
// decision.Route: rule | nlu | fallback
// decision.ShouldCallLLM tells whether to continue into LLM
```

## Notes

1. `chatterbot-corpus` is mostly chitchat; business intents still need curated data.
2. Multilingual bundle is a packaging format, not one fused multilingual classifier.
3. Keep thresholds and split seed stable for comparable offline evaluation.
4. Generated artifacts can grow quickly; plan storage strategy by environment.

## Commands Summary

```bash
make test
make eval
./scripts/train_chatterbot_models.sh
./scripts/feedback_loop.sh --input <feedback.csv>
GOWORK=off go test ./...
```
