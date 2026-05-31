# intent-nlu

Lightweight, embeddable intent classification engine for Go.

- Module: `github.com/godeps/intent-nlu`
- Chinese docs: `README.zh-CN.md`
- Core stack:
  - Tokenization: `github.com/go-ego/gse` (Chinese), normalized splitter for non-CJK
  - Classifier: `github.com/jbrukh/bayesian`

## What It Provides

1. Low-latency pre-LLM intent recognition (~1ms/request).
2. Skill routing for creative pipelines (video, image, audio, 3D, analysis).
3. Deterministic train/val/test evaluation pipeline.
4. Per-intent threshold calibration (optional).
5. Intent taxonomy normalization (aliases -> canonical intents).
6. Multi-language routing (`zh`, `en`, extensible).
7. Hybrid policy (`rules -> NLU -> fallback/LLM`).
8. Data feedback loop for active dataset improvement.
9. Reproducibility metadata in model meta and bundle manifest.

## Repository Layout

```text
intent-nlu/
  cmd/
    chat-nlu-train/            # train one language model
    chat-nlu-predict/          # predict by single model / model map / bundle
    chat-nlu-bundle/           # build multilingual bundle from trained models
    chat-nlu-feedback/         # feedback ingestion and dataset/review update
  dataset/chatterbot/          # chatterbot corpus loader
  datasets/
    default/
      zh_business.csv          # business intents (calendar, weather, greeting)
      en_business.csv
      zh_skill_routing.csv     # skill routing intents (creative, analysis, chat)
      en_skill_routing.csv
    generated/
      *_train.csv              # effective training samples
      *_file_map.yaml          # auto-generated chatterbot mappings
      eval/*.json              # evaluation/model meta snapshots
    feedback/
      review/                  # low-confidence/unknown queue
      archive/                 # optional archived feedback
  docs/
    architecture.md
    skill-routing-integration.md
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
  embedded_bundle.go
```

## Quick Start

### 1) Run tests

```bash
make test
```

### 2) Reproducible local evaluation (`make eval`)

```bash
make eval
```

Outputs:

- `datasets/generated/eval/zh_eval.json`
- `datasets/generated/eval/en_eval.json`

Both reports include split metrics (accuracy, macro-F1, micro-F1, confusion matrix, per-intent metrics) and training metadata.

### 3) One-click corpus training (`zh,en` default)

```bash
make train
```

Outputs:

1. `models/model-zh`, `models/model-en`
2. `models/multilingual`
3. `datasets/generated/{zh,en}_train.csv`
4. `datasets/generated/eval/{zh,en}_meta.json`

## Intent Classes

Default embedded models include business, skill routing, and chitchat intents:

- `zh`: 26 classes
- `en`: 30 classes
- fallback: `unknown`

### Skill Routing Intents

| Intent | Description | Saker Skill | ZH Samples | EN Samples |
| --- | --- | --- | --- | --- |
| `creative_video` | Video production, editing, TVC, ads | `kanban-video-orchestrator` | 200 | 200 |
| `creative_image` | Image generation, poster, illustration | `generate_image` | 200 | 203 |
| `creative_audio` | Music, sound effects, audio production | `generate_music` | 100 | 100 |
| `creative_3d` | 3D modeling, rendering | `generate_3d` | 80 | 80 |
| `media_analysis` | Video/image understanding, description | `analyze_video` | 100 | 100 |
| `general_chat` | Chitchat, questions, non-creative tasks | (no skill) | 300 | 300 |

### Business Intents

| Intent | Description | zh | en |
| --- | --- | --- | --- |
| `calendar_info` | Date, weekday, holiday, lunar/solar calendar queries | Yes | Yes |
| `weather_info` | Weather forecast, rain, temperature | Yes | Yes |
| `chitchat_greeting` | Direct short greetings (hi/hello/good morning) | Yes | Yes |

### Chitchat Intents

| Intent | Description | zh | en |
| --- | --- | --- | --- |
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

### Taxonomy Aliases

Applied at inference time via `NormalizeIntent()`:

```go
"video_production"  -> "creative_video"
"video_editing"     -> "creative_video"
"film_production"   -> "creative_video"
"ad_production"     -> "creative_video"
"image_generation"  -> "creative_image"
"poster_design"     -> "creative_image"
"music_creation"    -> "creative_audio"
"audio_production"  -> "creative_audio"
"3d_modeling"       -> "creative_3d"
"video_analysis"    -> "media_analysis"
"image_analysis"    -> "media_analysis"
```

### Supported Languages

| Language | Code | Default Embedded Model | Auto Detect | Notes |
| --- | --- | --- | --- | --- |
| Chinese | `zh` | Yes | Yes | `gse` tokenizer, 26 classes |
| English | `en` | Yes | Yes | normalized tokenizer, 30 classes |
| Japanese | `ja` | No (train yourself) | Yes | language detection supported |
| Korean | `ko` | No (train yourself) | Yes | language detection supported |

### Recommended Confidence Thresholds

| Use Case | Suggested Threshold | Behavior |
| --- | --- | --- |
| Strict skill routing (high precision) | `0.75 - 0.85` | Only high-confidence intents accepted; fallback increases |
| Business routing (balanced) | `0.60 - 0.70` | Good default for production intent dispatch |
| Default baseline | `0.55` | Current training default if no per-intent override |
| Recall-first exploration | `0.40 - 0.55` | More matched intents, but more false positives |
| Uncertain intent | `< threshold` => `unknown` | Route to fallback or LLM |

### Operational Notes

| Topic | Risk | Recommendation |
| --- | --- | --- |
| Fine-grained classes | Confusion across similar chitchat intents | Keep enough per-intent samples and evaluate confusion matrix |
| Corpus bias | chatterbot data is mostly chitchat | Always mix business/skill routing CSV for production tasks |
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
3. Merge business CSV (`<lang>_business.csv`) and skill routing CSV (`<lang>_skill_routing.csv`).
4. Train models with split/evaluation/calibration.
5. Build multilingual bundle via `cmd/chat-nlu-bundle`.

### B) Manual training CLI

```bash
GOWORK=off go run ./cmd/chat-nlu-train \
  -lang zh \
  -corpus-root /path/to/chatterbot_corpus/data/chinese \
  -file-map ./examples/file_intent_map.yaml \
  -extra-csv ./datasets/default/zh_business.csv,./datasets/default/zh_skill_routing.csv \
  -dump-samples ./datasets/generated/zh_train.csv \
  -eval-report ./datasets/generated/eval/zh_meta.json \
  -out ./models/model-zh \
  -version 2026.05.31.zh.1 \
  -threshold 0.55 \
  -split-enabled=true \
  -train-ratio 0.8 \
  -val-ratio 0.1 \
  -test-ratio 0.1 \
  -seed 42 \
  -auto-calibrate-thresholds=true
```

Important flags:

- Data: `-corpus-root`, `-file-map`, `-category-map`, `-extra-csv` (comma-separated multi-file)
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
  -version 2026.05.31.bundle.1 \
  -corpus-repo-url https://github.com/gunthercox/chatterbot-corpus.git \
  -corpus-commit <commit> \
  -training-params "seed=42,train_ratio=0.8,val_ratio=0.1,test_ratio=0.1"
```

## Prediction

### Single model

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -model ./models/model-zh \
  -text "帮我做个产品视频" \
  -lang auto \
  -topk 3
```

### Multi-model map

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -models "zh=./models/model-zh,en=./models/model-en" \
  -text "create a poster" \
  -lang auto
```

### Bundle

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -bundle ./models/multilingual \
  -text "做一个3D模型" \
  -lang auto
```

### No model flags (use embedded default bundle)

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -text "analyze this video" \
  -lang auto
```

If `-bundle`, `-models`, and `-model` are all omitted, the command loads embedded default models.

## Embedded Bundle (for dependency consumers)

`intent-nlu` embeds default multilingual models into the package.
When another Go service imports this module, it can load models without shipping external files.

```go
import chatnlu "github.com/godeps/intent-nlu"

router, err := chatnlu.NewRouterFromEmbedded()
if err != nil {
    panic(err)
}

pred, err := router.Predict(context.Background(), "帮我画一张海报", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "zh",
})
// pred.Intent == "creative_image", pred.Confidence == 0.96
```

Optional custom extraction cache directory:

```go
router, err := chatnlu.NewRouterFromEmbeddedIn("./.cache/intent-nlu")
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

pred, err := engine.Predict(context.Background(), "做一段背景音乐", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "auto",
})
// pred.Intent == "creative_audio"
```

### Router

```go
router, err := chatnlu.NewRouterFromBundle("./models/multilingual")
if err != nil {
    panic(err)
}

pred, err := router.Predict(context.Background(), "create a 3D model", chatnlu.PredictOptions{
    TopK:         3,
    LanguageHint: "auto",
})
// pred.Intent == "creative_3d"
```

### Hybrid Policy (rules -> NLU -> fallback)

```go
policy := &chatnlu.HybridPolicy{
    Router: router,
    Rules: []chatnlu.DeterministicRule{
        {ID: "r1", Intent: "video_production", ContainsAny: []string{"tvc", "宣传片制作"}},
    },
}
_ = policy.Prepare() // taxonomy normalizes: "video_production" -> "creative_video"

decision, err := policy.Decide(context.Background(), userText, chatnlu.PredictOptions{TopK: 3})
// decision.Route: rule | nlu | fallback
// decision.ShouldCallLLM tells whether to continue into LLM
```

## Notes

1. `chatterbot-corpus` is mostly chitchat; business and skill routing intents need curated data.
2. Multilingual bundle is a packaging format, not one fused multilingual classifier.
3. Keep thresholds and split seed stable for comparable offline evaluation.
4. Generated artifacts can grow quickly; plan storage strategy by environment.
5. See `docs/skill-routing-integration.md` for Saker integration details and evaluation results.

## Commands Summary

```bash
make test                                  # run all tests
make eval                                  # reproducible evaluation (CSV only)
make train                                 # full training + bundling
./scripts/feedback_loop.sh --input <csv>   # feedback data ingestion
```
