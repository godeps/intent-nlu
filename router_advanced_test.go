package chatnlu

import (
	"context"
	"testing"
)

func TestRouterPickBestLanguageByRawScore(t *testing.T) {
	zhDir := trainModelForTest(t, "zh", []Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
		{Text: "后天是几号", Intent: "calendar_info"},
		{Text: "上海明天天气", Intent: "weather_info"},
		{Text: "北京今天会下雨吗", Intent: "weather_info"},
	})
	enDir := trainModelForTest(t, "en", []Sample{
		{Text: "what day is tomorrow", Intent: "calendar_info"},
		{Text: "what date is next monday", Intent: "calendar_info"},
		{Text: "weather forecast tomorrow", Intent: "weather_info"},
		{Text: "will it rain in beijing", Intent: "weather_info"},
	})

	router, err := NewRouterFromDirs(map[string]string{
		"zh": zhDir,
		"en": enDir,
	}, "zh")
	if err != nil {
		t.Fatalf("NewRouterFromDirs failed: %v", err)
	}

	lang, ok := router.pickBestLanguageByRawScore(context.Background(), "weather tomorrow", PredictOptions{TopK: 2})
	if !ok {
		t.Fatalf("expected best language selection")
	}
	if lang != LanguageEN {
		t.Fatalf("expected en, got %s", lang)
	}

	lang, ok = router.pickBestLanguageByRawScore(context.Background(), "明天星期几", PredictOptions{TopK: 2})
	if !ok {
		t.Fatalf("expected best language selection")
	}
	if lang != LanguageZH {
		t.Fatalf("expected zh, got %s", lang)
	}
}

func TestRouterPredictWithForcedCrossLanguageFallback(t *testing.T) {
	zhDir := trainModelForTest(t, "zh", []Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
		{Text: "后天是几号", Intent: "calendar_info"},
		{Text: "上海明天天气", Intent: "weather_info"},
		{Text: "北京今天会下雨吗", Intent: "weather_info"},
	})
	enDir := trainModelForTest(t, "en", []Sample{
		{Text: "what day is tomorrow", Intent: "calendar_info"},
		{Text: "what date is next monday", Intent: "calendar_info"},
		{Text: "weather forecast tomorrow", Intent: "weather_info"},
		{Text: "will it rain in beijing", Intent: "weather_info"},
	})

	router := NewRouterWithOptions("zh", RouterOptions{
		AutoDetectMinConfidence:     1.1,
		ShortTextRuneLimit:          3,
		EnableCrossLanguageFallback: true,
	})
	if err := router.Load("zh", zhDir); err != nil {
		t.Fatalf("load zh failed: %v", err)
	}
	if err := router.Load("en", enDir); err != nil {
		t.Fatalf("load en failed: %v", err)
	}

	pred, err := router.Predict(context.Background(), "weather tomorrow", PredictOptions{
		TopK:         2,
		LanguageHint: "auto",
	})
	if err != nil {
		t.Fatalf("router predict failed: %v", err)
	}
	if pred.Language != "en" {
		t.Fatalf("expected en routed language, got %s", pred.Language)
	}
	if pred.Intent != "weather_info" {
		t.Fatalf("expected weather_info, got %+v", pred)
	}
}
