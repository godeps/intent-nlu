package chatnlu

import (
	"context"
	"testing"
)

func TestTrainSaveLoadPredict_ZH(t *testing.T) {
	samples := []Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
		{Text: "后天是什么日子", Intent: "calendar_info"},
		{Text: "今天农历几号", Intent: "calendar_info"},
		{Text: "元宵节是哪天", Intent: "calendar_info"},
		{Text: "上海明天天气", Intent: "weather_info"},
		{Text: "北京会下雨吗", Intent: "weather_info"},
		{Text: "今天气温多少", Intent: "weather_info"},
		{Text: "深圳天气预报", Intent: "weather_info"},
	}

	cfg := DefaultTrainConfig()
	cfg.Version = "test-v1"
	cfg.DefaultThreshold = 0.01
	cfg.Tokenizer.Language = "zh"

	model, err := Train(samples, cfg)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}
	if model.Meta().TrainingSampleCount == 0 {
		t.Fatalf("expected training samples > 0")
	}
	if model.Meta().Language != "zh" {
		t.Fatalf("expected model language zh, got %s", model.Meta().Language)
	}

	dir := t.TempDir()
	if err := model.SaveDir(dir); err != nil {
		t.Fatalf("SaveDir failed: %v", err)
	}

	engine, err := NewEngineFromDir(dir)
	if err != nil {
		t.Fatalf("NewEngineFromDir failed: %v", err)
	}

	pred1, err := engine.Predict(context.Background(), "明天是星期几", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Predict calendar failed: %v", err)
	}
	if !pred1.Matched || pred1.Intent != "calendar_info" {
		t.Fatalf("unexpected prediction for calendar: %+v", pred1)
	}
	if pred1.Language != "zh" {
		t.Fatalf("expected zh output language, got %s", pred1.Language)
	}

	pred2, err := engine.Predict(context.Background(), "北京今天会不会下雨", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Predict weather failed: %v", err)
	}
	if !pred2.Matched || pred2.Intent != "weather_info" {
		t.Fatalf("unexpected prediction for weather: %+v", pred2)
	}

	pred3, err := engine.Predict(context.Background(), "随便聊聊天", PredictOptions{TopK: 2, MinConfidence: 0.999})
	if err != nil {
		t.Fatalf("Predict unknown failed: %v", err)
	}
	if pred3.Matched || pred3.Intent != DefaultUnknownIntent {
		t.Fatalf("expected unknown intent with high threshold, got %+v", pred3)
	}
}

func TestTrainSaveLoadPredict_EN(t *testing.T) {
	samples := []Sample{
		{Text: "what day is tomorrow", Intent: "calendar_info"},
		{Text: "what is the date today", Intent: "calendar_info"},
		{Text: "is tomorrow a holiday", Intent: "calendar_info"},
		{Text: "weather in shanghai tomorrow", Intent: "weather_info"},
		{Text: "will it rain in beijing", Intent: "weather_info"},
		{Text: "today weather forecast", Intent: "weather_info"},
	}

	cfg := DefaultTrainConfig()
	cfg.Version = "test-en-v1"
	cfg.DefaultThreshold = 0.01
	cfg.Tokenizer.Language = "en"

	model, err := Train(samples, cfg)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}
	if model.Meta().Language != "en" {
		t.Fatalf("expected model language en, got %s", model.Meta().Language)
	}

	dir := t.TempDir()
	if err := model.SaveDir(dir); err != nil {
		t.Fatalf("SaveDir failed: %v", err)
	}

	engine, err := NewEngineFromDir(dir)
	if err != nil {
		t.Fatalf("NewEngineFromDir failed: %v", err)
	}

	pred, err := engine.Predict(context.Background(), "will it rain in beijing tomorrow", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if !pred.Matched || pred.Intent != "weather_info" {
		t.Fatalf("unexpected prediction for english weather: %+v", pred)
	}
	if pred.Language != "en" {
		t.Fatalf("expected en output language, got %s", pred.Language)
	}
}
