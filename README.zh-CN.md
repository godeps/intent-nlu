# chat-nlu（中文文档）

一个可复用的 Go 轻量意图识别工具包。

- 分词：
中文使用 `github.com/go-ego/gse`，其他语言使用轻量分词器。
- 分类：
`github.com/jbrukh/bayesian`
- 模块名：
`github.com/godeps/chat-nlu`

English documentation:
`README.md`

## 解决的问题

1. 在 LLM 前做低延迟意图识别。
2. 支持多语言模型路由（当前已验证 `zh`、`en`）。
3. 支持从 `chatterbot-corpus` 一键下载并训练。
4. 支持多语言 bundle 打包与加载。

## 仓库目录结构

```text
chat-nlu/
  cmd/
    chat-nlu-train/            # 训练 CLI（单语言模型）
    chat-nlu-predict/          # 预测 CLI（单模型/多模型/bundle）
  dataset/
    chatterbot/                # chatterbot 语料加载器
  datasets/
    default/                   # 默认业务训练数据（已提交）
      zh_business.csv
      en_business.csv
    generated/                 # 训练生成数据（有效训练集与映射）
      *.csv
      *_file_map.yaml
  docs/
    architecture.md
  examples/
    file_intent_map.yaml
  models/
    model-zh/                  # 中文模型
      model.gob
      meta.json
    model-en/                  # 英文模型
      model.gob
      meta.json
    multilingual/              # 多语言 bundle
      manifest.json
      models/
        zh/
          model.gob
          meta.json
        en/
          model.gob
          meta.json
  scripts/
    train_chatterbot_models.sh # 一键下载语料+训练+打包

  # 核心包文件
  types.go
  language.go
  tokenizer.go
  trainer.go
  model.go
  engine.go
  router.go
  router_bundle.go
```

## 核心概念

1. 单模型：
一个语言一个模型目录（`model.gob` + `meta.json`）。
2. 多模型路由：
按语言路由到不同模型目录。
3. Bundle：
把多个语言模型按标准目录合并，统一通过 `manifest.json` 加载。

## 安装

```bash
go get github.com/godeps/chat-nlu
```

## 快速开始

### 1) 一键训练（默认 `zh,en`）

```bash
cd ../godeps/chat-nlu
./scripts/train_chatterbot_models.sh
```

默认输出：

1. 模型：
`./models/model-zh`、`./models/model-en`
2. 多语言 bundle：
`./models/multilingual`
3. 有效训练集：
`./datasets/generated/zh_train.csv`、`./datasets/generated/en_train.csv`

### 2) 使用 bundle 预测

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -bundle ./models/multilingual \
  -text "明天星期几" \
  -lang auto
```

### 3) 使用单模型预测

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -model ./models/model-zh \
  -text "你好" \
  -lang auto
```

## 训练方法

## 方法 A：一键脚本（推荐）

脚本：
`./scripts/train_chatterbot_models.sh`

脚本流程：

1. 拉取或更新 `https://github.com/gunthercox/chatterbot-corpus`。
2. 按语言目录加载语料。
3. 自动生成文件名到意图映射（`chitchat_<filename>`）。
4. 自动追加 `datasets/default/<lang>_business.csv`（如果存在）。
5. 训练每个语言模型。
6. 导出有效训练集到 `datasets/generated`。
7. 生成 `models/multilingual` bundle。

常用参数示例：

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

关键参数说明：

1. `--langs`：
训练语言列表，逗号分隔。
2. `--threshold`：
默认置信度阈值。
3. `--default-extra-dir`：
默认业务数据目录（按 `<lang>_business.csv` 命名）。
4. `--generated-dataset-dir`：
有效训练集导出目录。
5. `--merge-bundle`：
是否生成多语言 bundle。
6. `--router-default-lang`：
bundle 默认路由语言。

## 方法 B：手动 CLI 训练

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

训练参数摘要：

1. `-lang`：
模型语言。
2. `-corpus-root`：
chatterbot 语料目录。
3. `-file-map`、`-category-map`：
语料映射配置。
4. `-default-intent`：
未映射文件的默认意图。
5. `-skip-unmapped`、`-include-replies`：
语料加载策略。
6. `-extra-csv`：
补充业务样本（`text,intent`）。
7. `-dump-samples`：
导出最终训练数据。
8. `-threshold`、`-thresholds`：
阈值配置。
9. `-out`、`-version`：
模型输出目录与版本号。

## 预测方法

## 方法 A：单模型预测

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -model ./models/model-zh \
  -text "明天星期几" \
  -lang auto \
  -topk 3
```

## 方法 B：多模型映射预测

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -models "zh=./models/model-zh,en=./models/model-en" \
  -text "what is the weather tomorrow" \
  -lang auto
```

## 方法 C：bundle 预测

```bash
GOWORK=off go run ./cmd/chat-nlu-predict \
  -bundle ./models/multilingual \
  -text "hello" \
  -lang auto
```

预测参数摘要：

1. `-model`：
单模型目录。
2. `-models`：
多模型映射（`lang=path`）。
3. `-bundle`：
bundle 目录（必须包含 `manifest.json`）。
4. `-lang`：
语言提示（`auto/zh/en/...`）。
5. `-min-confidence`：
单次请求阈值覆盖。
6. `-topk`：
返回候选数量。

## 代码接入示例

单模型：

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

多模型路由：

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

Bundle 加载：

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

## 注意事项

1. `chatterbot-corpus` 主要是闲聊语料。
业务意图（如 `calendar_info`、`weather_info`）建议补充自有标注数据。
2. 阈值需要调优。
阈值高会提升精度但增加 `unknown`，阈值低会提升召回但误判上升。
3. bundle 是“多模型打包”，不是“单一融合分类器”。
每种语言仍然独立预测。
4. 建议对训练数据和模型进行版本管理。
保证可复现性（语料版本 + 映射 + 阈值）。
5. 生成数据和模型会增加仓库体积。
如有需要可迁移到制品仓库存储。
6. 本地缓存目录是 `./.cache`。
已在 `.gitignore` 中忽略，可随时重建。

## 常见问题

1. 语料 YAML 解析报错：
更新到最新代码后重试（已兼容 chatterbot 常见混合结构）。
2. 结果总是 `unknown`：
先用 `-min-confidence` 降低阈值排查，再调整训练阈值。
3. 语言路由不准：
显式传 `-lang` 或设置合适默认路由语言。
4. 业务意图命中率低：
检查 `file-map` 和业务 CSV 样本质量与覆盖度。

## 测试

```bash
cd ../godeps/chat-nlu
GOWORK=off go test ./...
```
