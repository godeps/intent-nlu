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

func TestTrainPredictSkillRouting_ZH(t *testing.T) {
	samples := []Sample{
		{Text: "帮我制作一个产品宣传视频", Intent: "creative_video"},
		{Text: "做一个30秒的广告片", Intent: "creative_video"},
		{Text: "生成一段短视频", Intent: "creative_video"},
		{Text: "帮我做个vlog", Intent: "creative_video"},
		{Text: "来个宣传片", Intent: "creative_video"},
		{Text: "帮我画一张产品海报", Intent: "creative_image"},
		{Text: "生成一张宣传图片", Intent: "creative_image"},
		{Text: "设计一个logo给我", Intent: "creative_image"},
		{Text: "做个banner图", Intent: "creative_image"},
		{Text: "画一张插画", Intent: "creative_image"},
		{Text: "帮我配一段背景音乐", Intent: "creative_audio"},
		{Text: "生成一段BGM", Intent: "creative_audio"},
		{Text: "做一首歌曲", Intent: "creative_audio"},
		{Text: "制作一段音效", Intent: "creative_audio"},
		{Text: "帮我做一个3D产品模型", Intent: "creative_3d"},
		{Text: "生成一个三维渲染图", Intent: "creative_3d"},
		{Text: "做一个3D场景", Intent: "creative_3d"},
		{Text: "帮我建模一个角色", Intent: "creative_3d"},
		{Text: "帮我分析一下这个视频的内容", Intent: "media_analysis"},
		{Text: "看看这张图片里有什么", Intent: "media_analysis"},
		{Text: "识别这个图片中的文字", Intent: "media_analysis"},
		{Text: "帮我描述一下这张照片", Intent: "media_analysis"},
		{Text: "今天天气怎么样", Intent: "general_chat"},
		{Text: "帮我写段代码", Intent: "general_chat"},
		{Text: "什么是机器学习", Intent: "general_chat"},
		{Text: "你好啊", Intent: "general_chat"},
		{Text: "推荐几本好书", Intent: "general_chat"},
	}

	cfg := DefaultTrainConfig()
	cfg.Version = "test-skill-zh"
	cfg.DefaultThreshold = 0.01
	cfg.Tokenizer.Language = "zh"

	model, err := Train(samples, cfg)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}

	dir := t.TempDir()
	if err := model.SaveDir(dir); err != nil {
		t.Fatalf("SaveDir failed: %v", err)
	}

	engine, err := NewEngineFromDir(dir)
	if err != nil {
		t.Fatalf("NewEngineFromDir failed: %v", err)
	}

	cases := []struct {
		text   string
		intent string
	}{
		{"帮我做个产品视频", "creative_video"},
		{"画一张海报", "creative_image"},
		{"帮我配个背景音乐", "creative_audio"},
		{"做一个3D模型", "creative_3d"},
		{"分析一下这个视频内容", "media_analysis"},
	}
	for _, tc := range cases {
		pred, err := engine.Predict(context.Background(), tc.text, PredictOptions{TopK: 3})
		if err != nil {
			t.Fatalf("Predict(%q) failed: %v", tc.text, err)
		}
		if !pred.Matched || pred.Intent != tc.intent {
			t.Errorf("Predict(%q) = %q (matched=%v), want %q", tc.text, pred.Intent, pred.Matched, tc.intent)
		}
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

func TestTrainPredictSkillRouting_EN(t *testing.T) {
	samples := []Sample{
		{Text: "create a product promotional video", Intent: "creative_video"},
		{Text: "make me a 30 second commercial", Intent: "creative_video"},
		{Text: "generate a short video ad", Intent: "creative_video"},
		{Text: "produce a brand video", Intent: "creative_video"},
		{Text: "make a vlog for me", Intent: "creative_video"},
		{Text: "design a product poster for me", Intent: "creative_image"},
		{Text: "generate a promotional image", Intent: "creative_image"},
		{Text: "create a logo design", Intent: "creative_image"},
		{Text: "make a banner graphic", Intent: "creative_image"},
		{Text: "draw an illustration", Intent: "creative_image"},
		{Text: "create background music for me", Intent: "creative_audio"},
		{Text: "generate a BGM track", Intent: "creative_audio"},
		{Text: "make me a song", Intent: "creative_audio"},
		{Text: "produce a sound effect", Intent: "creative_audio"},
		{Text: "create a 3D product model", Intent: "creative_3d"},
		{Text: "generate a three dimensional render", Intent: "creative_3d"},
		{Text: "build a 3D scene", Intent: "creative_3d"},
		{Text: "make a 3D character model", Intent: "creative_3d"},
		{Text: "analyze the content of this video", Intent: "media_analysis"},
		{Text: "tell me what is in this image", Intent: "media_analysis"},
		{Text: "recognize the text in this picture", Intent: "media_analysis"},
		{Text: "describe this photo for me", Intent: "media_analysis"},
		{Text: "what is the weather like today", Intent: "general_chat"},
		{Text: "help me write some code", Intent: "general_chat"},
		{Text: "what is machine learning", Intent: "general_chat"},
		{Text: "hello there", Intent: "general_chat"},
		{Text: "recommend some good books", Intent: "general_chat"},
	}

	cfg := DefaultTrainConfig()
	cfg.Version = "test-skill-en"
	cfg.DefaultThreshold = 0.01
	cfg.Tokenizer.Language = "en"

	model, err := Train(samples, cfg)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}

	dir := t.TempDir()
	if err := model.SaveDir(dir); err != nil {
		t.Fatalf("SaveDir failed: %v", err)
	}

	engine, err := NewEngineFromDir(dir)
	if err != nil {
		t.Fatalf("NewEngineFromDir failed: %v", err)
	}

	cases := []struct {
		text   string
		intent string
	}{
		{"create a product video", "creative_video"},
		{"design a poster for me", "creative_image"},
		{"make background music", "creative_audio"},
		{"create a 3D model", "creative_3d"},
		{"analyze this video", "media_analysis"},
	}
	for _, tc := range cases {
		pred, err := engine.Predict(context.Background(), tc.text, PredictOptions{TopK: 3})
		if err != nil {
			t.Fatalf("Predict(%q) failed: %v", tc.text, err)
		}
		if !pred.Matched || pred.Intent != tc.intent {
			t.Errorf("Predict(%q) = %q (matched=%v), want %q", tc.text, pred.Intent, pred.Matched, tc.intent)
		}
	}
}
