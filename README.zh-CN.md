# intent-nlu（中文文档）

轻量、可嵌入的 Go 意图分类引擎。

- 模块名：`github.com/godeps/intent-nlu`
- 英文文档：`README.md`
- 核心技术：
  - 分词：`github.com/go-ego/gse`（中文）+ 非 CJK 轻量分词
  - 分类：`github.com/jbrukh/bayesian`

## 能力概览

1. LLM 前的低延迟意图识别（~1ms/次）。
2. 创意技能路由（视频、图片、音频、3D、分析）。
3. 可复现的 train/val/test 评估流程。
4. 按意图阈值自动校准（可选）。
5. 意图别名归一（taxonomy）。
6. 多语言路由（`zh`、`en`，可扩展）。
7. 混合策略（`rule -> NLU -> fallback/LLM`）。
8. 反馈数据回流（active learning）。
9. 模型和 bundle 级可追溯元数据。

## 目录结构

```text
intent-nlu/
  cmd/
    chat-nlu-train/            # 单语言训练
    chat-nlu-predict/          # 单模型/多模型/bundle 预测
    chat-nlu-bundle/           # 多语言 bundle 打包
    chat-nlu-feedback/         # 反馈数据回流
  dataset/chatterbot/          # chatterbot 语料加载器
  datasets/
    default/
      zh_business.csv          # 业务意图（日历、天气、问候）
      en_business.csv
      zh_skill_routing.csv     # 技能路由意图（创意、分析、闲聊）
      en_skill_routing.csv
    generated/
      *_train.csv
      *_file_map.yaml
      eval/*.json
    feedback/
      review/
      archive/
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

  # 核心包
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

## 快速开始

### 1) 运行测试

```bash
make test
```

### 2) 可复现评估（`make eval`）

```bash
make eval
```

输出：

- `datasets/generated/eval/zh_eval.json`
- `datasets/generated/eval/en_eval.json`

报告包含：accuracy、macro-F1、micro-F1、混淆矩阵、按意图指标、训练元数据。

### 3) 一键训练（默认 `zh,en`）

```bash
make train
```

输出：

1. `models/model-zh`、`models/model-en`
2. `models/multilingual`
3. `datasets/generated/{zh,en}_train.csv`
4. `datasets/generated/eval/{zh,en}_meta.json`

## 意图分类

当前默认内嵌模型包含业务、技能路由和闲聊意图：

- `zh`：26 类
- `en`：30 类
- 兜底：`unknown`

### 技能路由意图

| 意图 | 说明 | Saker 技能 | 中文样本 | 英文样本 |
| --- | --- | --- | --- | --- |
| `creative_video` | 视频制作、剪辑、TVC、广告 | `kanban-video-orchestrator` | 200 | 200 |
| `creative_image` | 图片生成、海报、插画 | `generate_image` | 200 | 203 |
| `creative_audio` | 音乐、音效、音频制作 | `generate_music` | 100 | 100 |
| `creative_3d` | 3D 建模、渲染 | `generate_3d` | 80 | 80 |
| `media_analysis` | 视频/图片理解、描述 | `analyze_video` | 100 | 100 |
| `general_chat` | 闲聊、提问、非创意任务 | （无技能） | 300 | 300 |

### 业务意图

| 意图 | 说明 | zh | en |
| --- | --- | --- | --- |
| `calendar_info` | 日期、星期、节假日、农历/公历查询 | 是 | 是 |
| `weather_info` | 天气预报、降雨、温度 | 是 | 是 |
| `chitchat_greeting` | 直接问候（你好/早上好/hello） | 是 | 是 |

### 闲聊意图

| 意图 | 说明 | zh | en |
| --- | --- | --- | --- |
| `chitchat_greetings` | 问候变体（来自语料细分） | 是 | 是 |
| `chitchat_ai` | AI/助手相关闲聊 | 是 | 是 |
| `chitchat_botprofile` | 机器人身份/能力/偏好 | 是 | 是 |
| `chitchat_conversations` | 开放域泛化闲聊 | 是 | 是 |
| `chitchat_emotion` | 情绪、安慰、心情类闲聊 | 是 | 是 |
| `chitchat_food` | 美食饮品相关闲聊 | 是 | 是 |
| `chitchat_gossip` | 八卦/轻社交话题 | 是 | 是 |
| `chitchat_history` | 历史类闲聊 | 是 | 是 |
| `chitchat_humor` | 笑话、幽默内容 | 是 | 是 |
| `chitchat_literature` | 文学、写作、书籍闲聊 | 是 | 是 |
| `chitchat_money` | 金钱/理财轻话题 | 是 | 是 |
| `chitchat_movies` | 电影娱乐闲聊 | 是 | 是 |
| `chitchat_politics` | 政治社会闲聊 | 是 | 是 |
| `chitchat_psychology` | 心理、人格、认知类闲聊 | 是 | 是 |
| `chitchat_science` | 科学知识闲聊 | 是 | 是 |
| `chitchat_sports` | 体育闲聊 | 是 | 是 |
| `chitchat_trivia` | 泛知识问答/冷知识闲聊 | 是 | 是 |
| `chitchat_coding` | 编程开发类闲聊 | 否 | 是 |
| `chitchat_computers` | 计算机/设备/软件闲聊 | 否 | 是 |
| `chitchat_health` | 健康与生活方式闲聊 | 否 | 是 |
| `chitchat_tech_support` | 轻技术支持类闲聊 | 否 | 是 |

### Taxonomy 别名

在推理时通过 `NormalizeIntent()` 应用：

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

### 支持语言

| 语言 | 代码 | 默认内嵌模型 | 自动识别 | 说明 |
| --- | --- | --- | --- | --- |
| 中文 | `zh` | 是 | 是 | `gse` 分词，26 类 |
| 英文 | `en` | 是 | 是 | 轻量分词，30 类 |
| 日文 | `ja` | 否（需自行训练） | 是 | 支持语言识别 |
| 韩文 | `ko` | 否（需自行训练） | 是 | 支持语言识别 |

### 推荐置信度阈值

| 使用场景 | 推荐阈值 | 说明 |
| --- | --- | --- |
| 严格技能路由（高精度） | `0.75 - 0.85` | 只接受高置信度意图，兜底增加 |
| 业务路由（精度/召回平衡） | `0.60 - 0.70` | 生产环境常用区间 |
| 默认基线 | `0.55` | 当前训练默认阈值 |
| 召回优先 | `0.40 - 0.55` | 召回更高，误判上升 |
| 不确定输入 | `< 阈值 -> unknown` | 进入 fallback 或 LLM |

### 运行注意事项

| 主题 | 风险 | 建议 |
| --- | --- | --- |
| 细分类较多 | 相近闲聊类互相混淆 | 保证每类样本量，结合 confusion matrix 调整 |
| 语料偏置 | chatterbot 以闲聊为主 | 生产必须叠加业务/技能路由标注数据 |
| 多语言路由 | 超短文本/混合文本易误路由 | 关键链路建议传入 language hint |
| 阈值漂移 | 重训后置信度分布变化 | 每次发布对比 `eval/*.json` 并重校准 |
| 内嵌模型更新 | 更新模型需升级依赖重编译 | 固定版本并在发布说明中标注模型版本 |

## 训练方式

### A）一键脚本（推荐）

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

脚本流程：

1. 拉取/更新 `chatterbot-corpus`。
2. 自动生成文件到意图映射（`chitchat_<file>`）。
3. 合并业务 CSV（`<lang>_business.csv`）和技能路由 CSV（`<lang>_skill_routing.csv`）。
4. 执行分割训练、评估和阈值校准。
5. 通过 `cmd/chat-nlu-bundle` 构建多语言 bundle。

### B）手动训练 CLI

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

关键参数：

- 数据：`-corpus-root`、`-file-map`、`-category-map`、`-extra-csv`（逗号分隔多文件）
- 评估：`-split-enabled`、`-train-ratio`、`-val-ratio`、`-test-ratio`、`-seed`、`-eval-report`
- 阈值：`-threshold`、`-thresholds`、`-auto-calibrate-thresholds`
- 归一：`-disable-taxonomy`（默认 `true`）、`-taxonomy-aliases`
- 可追溯：`-source-name`、`-source-version`、`-source-revision`、`-source-repo-url`、`-source-commit`

## Bundle 打包 CLI

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

## 预测

### 单模型

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -model ./models/model-zh \
  -text "帮我做个产品视频" \
  -lang auto \
  -topk 3
```

### 多模型映射

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

### 不传模型参数（自动使用内嵌默认模型）

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -text "analyze this video" \
  -lang auto
```

## 依赖方内嵌模型（Embedded Bundle）

`intent-nlu` 将默认多语言模型内嵌在包内，依赖方无需携带模型文件。

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

也可以指定解压缓存目录：

```go
router, err := chatnlu.NewRouterFromEmbeddedIn("./.cache/intent-nlu")
```

## 反馈回流

用途：

1. 将人工确认样本追加到 `datasets/default/<lang>_business.csv`
2. 将低置信度或 `unknown` 样本写入 review 队列

```bash
./scripts/feedback_loop.sh --input ./tmp/feedback.csv
```

支持的 CSV 字段：

- 必填：`text`
- 可选别名：
  - 语言：`language`
  - 预测意图：`pred_intent|intent|predicted_intent`
  - 分数：`confidence|score`
  - 人工标签：`final_intent|human_intent|label`

## 代码接入示例

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

### Hybrid Policy（规则 -> NLU -> 兜底）

```go
policy := &chatnlu.HybridPolicy{
    Router: router,
    Rules: []chatnlu.DeterministicRule{
        {ID: "r1", Intent: "video_production", ContainsAny: []string{"tvc", "宣传片制作"}},
    },
}
_ = policy.Prepare() // taxonomy 归一: "video_production" -> "creative_video"

decision, err := policy.Decide(context.Background(), userText, chatnlu.PredictOptions{TopK: 3})
// decision.Route: rule | nlu | fallback
// decision.ShouldCallLLM 表示是否进入 LLM
```

## 注意事项

1. `chatterbot-corpus` 主要是闲聊语料，业务和技能路由意图需要补充自有标注数据。
2. 多语言 bundle 是"多模型打包"，不是"单模型融合"。
3. 固定阈值、分割比例和 seed，才能做可比离线评估。
4. 训练与模型产物会增长较快，建议按环境规划存储策略。
5. Saker 集成详情和评估结果见 `docs/skill-routing-integration.md`。

## 常用命令

```bash
make test                                  # 运行所有测试
make eval                                  # 可复现评估（仅 CSV）
make train                                 # 全量训练 + 打包
./scripts/feedback_loop.sh --input <csv>   # 反馈数据回流
```
