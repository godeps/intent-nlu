# intent-nlu Architecture

## Overview

`intent-nlu` is a lightweight intent classification engine focused on low-latency pre-LLM routing, skill activation, and reusable deployment.

Core stack:

- Chinese tokenization: `gse`
- Non-CJK tokenization: normalized splitter
- Classifier: `jbrukh/bayesian`

## Main Components

- `tokenizer.go`: language-aware tokenization and normalization.
- `language.go`: language normalization and lightweight language detection.
- `taxonomy.go`: intent alias normalization (taxonomy), including skill routing aliases.
- `evaluation.go`: deterministic split, threshold calibration, metrics and confusion matrix.
- `trainer.go`: supervised training and metadata generation.
- `model.go`: model persistence and loading (`model.gob`, `meta.json`).
- `engine.go`: single-model prediction and hot reload.
- `router.go`: multi-language routing with optional cross-language raw-score fallback.
- `router_bundle.go`: bundle manifest and bundle loader.
- `embedded_bundle.go`: embedded bundle extraction and router loading for dependency consumers.
- `hybrid_policy.go`: deterministic rules + NLU + fallback orchestration.
- `dataset_csv.go`: CSV dataset loading and saving.
- `dataset/chatterbot/loader.go`: chatterbot corpus loader.

## Intent Categories

### Skill Routing Intents
Activate specialized execution pipelines:
- `creative_video`, `creative_image`, `creative_audio`, `creative_3d` — creative production
- `media_analysis` — video/image understanding
- `general_chat` — no skill activation (fallback to LLM)

### Tool Routing Intents
Route user requests to saker builtin tools:
- `web_search` — web search, documentation lookup (tools: web_search, web_fetch, browser)
- `coding_assist` — code writing, debugging, testing, execution (tools: bash, edit, read, write, grep, glob)
- `task_management` — task/kanban CRUD operations (tools: task_*, kanban_*)
- `file_operation` — file download, read, write, save (tools: fetch_file, read, write)
- `knowledge_qa` — recall past decisions, memory queries (tools: memory_read, recall_context)
- `workflow_automation` — cron jobs, pipeline automation, scheduling (tools: workflow, cron, loop)
- `data_analysis` — metrics analysis, log analysis, statistics (tools: bash, canvas_table_write)
- `document_creation` — writing docs, guides, reports (tools: canvas_create_node, write, craft_knowledge)
- `translation` — text translation between languages
- `summarization` — summarize logs, reports, discussions (tools: video_summarizer)

### Business Intents
Domain-specific routing:
- `calendar_info`, `weather_info` — structured data queries
- `chitchat_greeting` — greeting detection

### Chatterbot Intents
Fine-grained chitchat classification from chatterbot corpus:
- 17 `chitchat_*` classes (zh), 21 `chitchat_*` classes (en)
- All normalized to `chitchat_general` via taxonomy at inference time

## Runtime Flow

### 1) Pre-LLM phase

1. Optional deterministic rule matching (keywords, regex).
2. NLU prediction (`Engine` or `Router`).
3. In direct-routing mode, accept only intents above threshold.
4. In candidate mode, return TopK ranked intents/tools and keep LLM/tool-planner final selection.
5. If no direct route is accepted, go fallback path with candidates attached.

### 2) Fallback/LLM phase

`intent-nlu` does not replace LLM generation; it decides when to skip/route/escalate.
For tool planning, NLU is usually a high-recall candidate generator rather than
the final authority.

### 3) Feedback phase

Prediction logs + human labels are fed into `cmd/intent-nlu-feedback`:

- high-quality labels -> business dataset (`datasets/default/*_business.csv`)
- low-confidence/unknown -> review queue (`datasets/feedback/review`)

## Training & Evaluation Flow

1. Load data from chatterbot corpus and/or CSV files (business + skill routing).
2. Optionally apply taxonomy normalization (disabled by default during training).
3. Tokenize samples.
4. Deterministically split into train/val/test (hash + seed).
5. Train bayesian classifier on train split.
6. Optional threshold calibration on validation split.
7. Evaluate train/val/test with:
   - accuracy
   - macro/micro F1
   - per-intent precision/recall/F1
   - per-intent Top1/Top3/Top5 candidate recall
   - confusion matrix
8. Save model and metadata.

### Multi-file CSV Support

The `-extra-csv` flag accepts comma-separated file paths, allowing multiple dataset files to be combined in a single training run:

```bash
-extra-csv ./datasets/default/zh_business.csv,./datasets/default/zh_skill_routing.csv,./datasets/default/zh_tools_routing.csv
```

The training script (`scripts/train_chatterbot_models.sh`) automatically discovers and loads all `{lang}_*.csv` files from the `datasets/default/` directory, including:
- `{lang}_business.csv` — business intents (calendar, weather, greeting)
- `{lang}_skill_routing.csv` — skill routing intents (creative, analysis, chat)
- `{lang}_tools_routing.csv` — tool routing intents (search, code, tasks, files, etc.)
- `{lang}_tools_boost.csv` — supplemental short-phrase samples
- `{lang}_tools_boost2.csv` — targeted samples for weak intents

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
4. Taxonomy normalization can reduce intent drift, but default is disabled during training to preserve fine-grained classes. Aliases are applied at inference time.
5. Skill/tool routing has two operating modes: direct execution prioritizes precision, while LLM/tool-planner handoff prioritizes TopK candidate recall.
6. Candidate mode intentionally tolerates lower precision because the downstream LLM or policy layer makes the final choice. Missing a plausible tool candidate is usually worse than offering one extra candidate.

## Integration Pattern

Recommended production chain:

1. `HybridPolicy` orchestrates all layers
2. Layer 1: NLU inference (`Router`) — primary, handles paraphrased/colloquial input
3. Layer 2: Deterministic rules — safety net for keyword matches
4. Layer 3: Candidate handoff — call LLM/tool planner with ranked `Prediction.Candidates`
5. Capture feedback and retrain periodically
