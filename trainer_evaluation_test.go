package intentnlu

import "testing"

func TestTrainWithSplitCalibrationAndTaxonomy(t *testing.T) {
	samples := []Sample{
		{Text: "明天星期几", Intent: "holiday_info"},
		{Text: "后天是几号", Intent: "date_info"},
		{Text: "今天农历几号", Intent: "calendar"},
		{Text: "元宵节是哪天", Intent: "holiday_info"},
		{Text: "中秋节是几月几号", Intent: "holiday_info"},
		{Text: "下周一是几号", Intent: "date_info"},
		{Text: "周末是哪天", Intent: "calendar"},
		{Text: "国庆节是哪天", Intent: "holiday_info"},
		{Text: "春节是哪天", Intent: "holiday_info"},
		{Text: "明年元旦是周几", Intent: "calendar"},

		{Text: "上海明天天气", Intent: "weather"},
		{Text: "北京今天会下雨吗", Intent: "forecast"},
		{Text: "广州多少度", Intent: "weather"},
		{Text: "深圳这周天气预报", Intent: "weather_forecast"},
		{Text: "杭州会不会刮风", Intent: "weather"},
		{Text: "成都天气怎么样", Intent: "forecast"},
		{Text: "南京周末天气", Intent: "weather"},
		{Text: "苏州今天温度", Intent: "forecast"},
		{Text: "武汉未来三天天气", Intent: "weather_forecast"},
		{Text: "天津今天冷不冷", Intent: "weather"},

		{Text: "你好", Intent: "chitchat_greetings"},
		{Text: "早上好", Intent: "greeting"},
		{Text: "晚上好", Intent: "greetings"},
		{Text: "哈喽", Intent: "chitchat_greetings"},
		{Text: "在吗", Intent: "chat"},
		{Text: "最近怎么样", Intent: "chitchat"},
		{Text: "聊聊天", Intent: "chitchat_conversations"},
		{Text: "讲个笑话", Intent: "chitchat_humor"},
		{Text: "你会做什么", Intent: "chitchat_botprofile"},
		{Text: "帮我放松一下", Intent: "chitchat_emotion"},
	}

	cfg := DefaultTrainConfig()
	cfg.Version = "eval-taxonomy-v1"
	cfg.Tokenizer.Language = "zh"
	cfg.Taxonomy.Enabled = true
	cfg.Split = DatasetSplitConfig{
		Enabled:    true,
		TrainRatio: 0.7,
		ValRatio:   0.2,
		TestRatio:  0.1,
		Seed:       17,
	}
	cfg.AutoCalibrateThresholds = true
	cfg.Source = SourceMetadata{
		Name:     "unit-test",
		Version:  "v1",
		Revision: "r1",
		RepoURL:  "https://example.invalid/repo.git",
		Commit:   "deadbeef",
	}

	model, err := Train(samples, cfg)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}
	meta := model.Meta()

	if meta.Training.TrainSampleCount == 0 || meta.Training.ValSampleCount == 0 || meta.Training.TestSampleCount == 0 {
		t.Fatalf("expected non-empty splits, got train=%d val=%d test=%d",
			meta.Training.TrainSampleCount, meta.Training.ValSampleCount, meta.Training.TestSampleCount)
	}
	if !meta.Training.Calibrated {
		t.Fatalf("expected calibrated thresholds")
	}
	if meta.Training.Seed != 17 {
		t.Fatalf("expected seed 17, got %d", meta.Training.Seed)
	}
	if !meta.Training.Config.TaxonomyEnabled {
		t.Fatalf("taxonomy should be enabled")
	}
	if meta.Source.Name != "unit-test" || meta.Source.Commit != "deadbeef" {
		t.Fatalf("source metadata not persisted: %+v", meta.Source)
	}

	if _, ok := meta.Evaluation["train"]; !ok {
		t.Fatalf("missing train evaluation")
	}
	if _, ok := meta.Evaluation["val"]; !ok {
		t.Fatalf("missing val evaluation")
	}
	if _, ok := meta.Evaluation["test"]; !ok {
		t.Fatalf("missing test evaluation")
	}
	if report := meta.Evaluation["val"]; report.Samples == 0 {
		t.Fatalf("val report samples should be > 0")
	}

	if _, ok := meta.Thresholds["calendar_info"]; !ok {
		t.Fatalf("missing canonical threshold calendar_info: %+v", meta.Thresholds)
	}
	if _, ok := meta.Thresholds["weather_info"]; !ok {
		t.Fatalf("missing canonical threshold weather_info: %+v", meta.Thresholds)
	}
	if _, ok := meta.Thresholds["chitchat_greeting"]; !ok {
		t.Fatalf("missing canonical threshold chitchat_greeting: %+v", meta.Thresholds)
	}
	if _, ok := meta.Thresholds["holiday_info"]; ok {
		t.Fatalf("alias threshold key should be normalized: %+v", meta.Thresholds)
	}
}
