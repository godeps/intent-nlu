package intentnlu

import (
	"context"
	"testing"
)

func TestRouterPredictMultiLanguage(t *testing.T) {
	zhDir := trainModelForTest(t, "zh", []Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
		{Text: "后天是什么日子", Intent: "calendar_info"},
		{Text: "上海明天天气", Intent: "weather_info"},
		{Text: "北京会下雨吗", Intent: "weather_info"},
	})
	enDir := trainModelForTest(t, "en", []Sample{
		{Text: "what day is tomorrow", Intent: "calendar_info"},
		{Text: "is tomorrow a holiday", Intent: "calendar_info"},
		{Text: "weather in shanghai tomorrow", Intent: "weather_info"},
		{Text: "will it rain in beijing", Intent: "weather_info"},
	})

	router, err := NewRouterFromDirs(map[string]string{
		"zh": zhDir,
		"en": enDir,
	}, "en")
	if err != nil {
		t.Fatalf("NewRouterFromDirs failed: %v", err)
	}

	predZH, err := router.Predict(context.Background(), "明天是星期几", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("router predict zh failed: %v", err)
	}
	if predZH.Language != "zh" {
		t.Fatalf("expected zh routed language, got %s", predZH.Language)
	}
	if predZH.Intent != "calendar_info" {
		t.Fatalf("expected calendar_info, got %+v", predZH)
	}

	predEN, err := router.Predict(context.Background(), "will it rain in beijing tomorrow", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("router predict en failed: %v", err)
	}
	if predEN.Language != "en" {
		t.Fatalf("expected en routed language, got %s", predEN.Language)
	}
	if predEN.Intent != "weather_info" {
		t.Fatalf("expected weather_info, got %+v", predEN)
	}

	predHint, err := router.Predict(context.Background(), "what's tomorrow", PredictOptions{TopK: 2, LanguageHint: "zh"})
	if err != nil {
		t.Fatalf("router predict with hint failed: %v", err)
	}
	if predHint.Language != "zh" {
		t.Fatalf("expected forced zh by hint, got %s", predHint.Language)
	}
}

func trainModelForTest(t *testing.T, lang string, samples []Sample) string {
	t.Helper()
	cfg := DefaultTrainConfig()
	cfg.Version = "test-" + lang
	cfg.DefaultThreshold = 0.01
	cfg.Tokenizer.Language = lang

	model, err := Train(samples, cfg)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}

	dir := t.TempDir()
	if err := model.SaveDir(dir); err != nil {
		t.Fatalf("SaveDir failed: %v", err)
	}
	return dir
}
