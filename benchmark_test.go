package intentnlu

import (
	"context"
	"testing"
)

var benchSamplesZH = []Sample{
	{Text: "明天星期几", Intent: "calendar_info"},
	{Text: "后天是什么日子", Intent: "calendar_info"},
	{Text: "今天农历几号", Intent: "calendar_info"},
	{Text: "元宵节是哪天", Intent: "calendar_info"},
	{Text: "上海明天天气", Intent: "weather_info"},
	{Text: "北京会下雨吗", Intent: "weather_info"},
	{Text: "今天气温多少", Intent: "weather_info"},
	{Text: "深圳天气预报", Intent: "weather_info"},
}

var benchSamplesEN = []Sample{
	{Text: "what day is tomorrow", Intent: "calendar_info"},
	{Text: "what is the date today", Intent: "calendar_info"},
	{Text: "is tomorrow a holiday", Intent: "calendar_info"},
	{Text: "weather in shanghai tomorrow", Intent: "weather_info"},
	{Text: "will it rain in beijing", Intent: "weather_info"},
	{Text: "today weather forecast", Intent: "weather_info"},
}

func benchEngine(b *testing.B, lang string, samples []Sample) *Engine {
	b.Helper()
	cfg := DefaultTrainConfig()
	cfg.Version = "bench"
	cfg.DefaultThreshold = 0.01
	cfg.Tokenizer.Language = lang
	model, err := Train(samples, cfg)
	if err != nil {
		b.Fatalf("Train failed: %v", err)
	}
	dir := b.TempDir()
	if err := model.SaveDir(dir); err != nil {
		b.Fatalf("SaveDir failed: %v", err)
	}
	engine, err := NewEngineFromDir(dir)
	if err != nil {
		b.Fatalf("NewEngineFromDir failed: %v", err)
	}
	return engine
}

func BenchmarkTokenize_ZH(b *testing.B) {
	tok, err := NewTokenizer(TokenizerConfig{
		Language:    "zh",
		SearchMode:  true,
		HMM:         true,
		Lowercase:   true,
		MinTokenLen: 1,
		StripPunct:  true,
	})
	if err != nil {
		b.Fatalf("NewTokenizer failed: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.Tokenize("北京明天会不会下雨")
	}
}

func BenchmarkTokenize_EN(b *testing.B) {
	tok, err := NewTokenizer(TokenizerConfig{
		Language:    "en",
		Lowercase:   true,
		MinTokenLen: 1,
		StripPunct:  true,
	})
	if err != nil {
		b.Fatalf("NewTokenizer failed: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.Tokenize("will it rain in beijing tomorrow")
	}
}

func BenchmarkPredict_ZH(b *testing.B) {
	engine := benchEngine(b, "zh", benchSamplesZH)
	ctx := context.Background()
	opts := PredictOptions{TopK: 3}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Predict(ctx, "北京明天会不会下雨", opts)
	}
}

func BenchmarkPredict_EN(b *testing.B) {
	engine := benchEngine(b, "en", benchSamplesEN)
	ctx := context.Background()
	opts := PredictOptions{TopK: 3}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Predict(ctx, "will it rain in beijing tomorrow", opts)
	}
}

func BenchmarkRouterPredict_Auto(b *testing.B) {
	zhEngine := benchEngine(b, "zh", benchSamplesZH)
	enEngine := benchEngine(b, "en", benchSamplesEN)
	router := NewRouter("en")
	router.mu.Lock()
	router.engines[LanguageZH] = zhEngine
	router.engines[LanguageEN] = enEngine
	router.mu.Unlock()

	ctx := context.Background()
	opts := PredictOptions{TopK: 3, LanguageHint: "auto"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			router.Predict(ctx, "北京明天天气", opts)
		} else {
			router.Predict(ctx, "will it rain tomorrow", opts)
		}
	}
}

func BenchmarkTrain(b *testing.B) {
	cfg := DefaultTrainConfig()
	cfg.Version = "bench"
	cfg.DefaultThreshold = 0.01
	cfg.Tokenizer.Language = "zh"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Train(benchSamplesZH, cfg)
	}
}

func BenchmarkNormalizeIntent(b *testing.B) {
	aliases := DefaultIntentAliases()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NormalizeIntent("chitchat_greetings", aliases)
		NormalizeIntent("weather_forecast", aliases)
		NormalizeIntent("unknown_custom_intent", aliases)
	}
}

func BenchmarkDefaultIntentAliases(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DefaultIntentAliases()
	}
}

func BenchmarkDefaultIntentAliasesRef(b *testing.B) {
	for i := 0; i < b.N; i++ {
		defaultIntentAliasesRef()
	}
}
