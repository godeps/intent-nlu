package intentnlu

import (
	"context"
	"testing"
)

func TestHybridPolicySkillRouting(t *testing.T) {
	modelDir := trainModelForTest(t, "zh", []Sample{
		{Text: "帮我制作一个产品宣传视频", Intent: "creative_video"},
		{Text: "做一个30秒的广告片", Intent: "creative_video"},
		{Text: "生成一段短视频", Intent: "creative_video"},
		{Text: "来个宣传片", Intent: "creative_video"},
		{Text: "帮我画一张海报", Intent: "creative_image"},
		{Text: "设计一个logo", Intent: "creative_image"},
		{Text: "生成一张插画", Intent: "creative_image"},
		{Text: "做一张宣传图", Intent: "creative_image"},
		{Text: "今天天气怎么样", Intent: "general_chat"},
		{Text: "帮我写段代码", Intent: "general_chat"},
		{Text: "你好啊", Intent: "general_chat"},
		{Text: "推荐一本书", Intent: "general_chat"},
	})
	engine, err := NewEngineFromDir(modelDir)
	if err != nil {
		t.Fatalf("NewEngineFromDir failed: %v", err)
	}

	policy := &HybridPolicy{
		Engine: engine,
		Rules: []DeterministicRule{
			{
				ID:          "rule_video_keyword",
				Intent:      "video_production",
				ContainsAny: []string{"tvc", "宣传片制作"},
			},
		},
	}
	if err := policy.Prepare(); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	dec, err := policy.Decide(context.Background(), "帮我做一个TVC宣传片制作", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Decide(rule) failed: %v", err)
	}
	if dec.Route != HybridRouteRule {
		t.Fatalf("expected rule route, got %s", dec.Route)
	}
	if dec.Intent != "creative_video" {
		t.Fatalf("expected taxonomy-normalized creative_video, got %s", dec.Intent)
	}

	dec, err = policy.Decide(context.Background(), "帮我做一个产品广告片", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Decide(nlu) failed: %v", err)
	}
	if dec.Route != HybridRouteNLU {
		t.Fatalf("expected nlu route, got %s", dec.Route)
	}
	if dec.Intent != "creative_video" {
		t.Fatalf("expected creative_video via NLU, got %s", dec.Intent)
	}

	dec, err = policy.Decide(context.Background(), "画一张海报", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Decide(nlu image) failed: %v", err)
	}
	if dec.Route != HybridRouteNLU || dec.Intent != "creative_image" {
		t.Fatalf("expected creative_image via NLU, got route=%s intent=%s", dec.Route, dec.Intent)
	}

	dec, err = policy.Decide(context.Background(), "完全无关的随机测试输入内容", PredictOptions{
		TopK:          2,
		MinConfidence: 0.999,
	})
	if err != nil {
		t.Fatalf("Decide(fallback) failed: %v", err)
	}
	if dec.Route != HybridRouteFallback || !dec.ShouldCallLLM {
		t.Fatalf("expected fallback route, got route=%s shouldCallLLM=%v", dec.Route, dec.ShouldCallLLM)
	}
}

func TestHybridPolicyRuleNLUAndFallback(t *testing.T) {
	modelDir := trainModelForTest(t, "zh", []Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
		{Text: "后天是几号", Intent: "calendar_info"},
		{Text: "上海明天天气", Intent: "weather_info"},
		{Text: "北京今天会下雨吗", Intent: "weather_info"},
	})
	engine, err := NewEngineFromDir(modelDir)
	if err != nil {
		t.Fatalf("NewEngineFromDir failed: %v", err)
	}

	policy := &HybridPolicy{
		Engine: engine,
		Rules: []DeterministicRule{
			{
				ID:          "rule_refund",
				Intent:      "support_refund",
				ContainsAny: []string{"退款", "退费"},
			},
		},
	}
	if err := policy.Prepare(); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	dec, err := policy.Decide(context.Background(), "我要退款", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Decide(rule) failed: %v", err)
	}
	if dec.Route != HybridRouteRule || dec.Intent != "support_refund" || dec.ShouldCallLLM {
		t.Fatalf("unexpected rule decision: %+v", dec)
	}

	dec, err = policy.Decide(context.Background(), "明天是星期几", PredictOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Decide(nlu) failed: %v", err)
	}
	if dec.Route != HybridRouteNLU || dec.Intent != "calendar_info" || dec.ShouldCallLLM {
		t.Fatalf("unexpected nlu decision: %+v", dec)
	}

	dec, err = policy.Decide(context.Background(), "随便聊点没见过的话题", PredictOptions{
		TopK:          2,
		MinConfidence: 0.999,
	})
	if err != nil {
		t.Fatalf("Decide(fallback) failed: %v", err)
	}
	if dec.Route != HybridRouteFallback || dec.Intent != DefaultUnknownIntent || !dec.ShouldCallLLM {
		t.Fatalf("unexpected fallback decision: %+v", dec)
	}

	dec, err = policy.Decide(context.Background(), "明天是星期几", PredictOptions{
		TopK:          2,
		MinConfidence: 0.999,
		CandidateMode: true,
	})
	if err != nil {
		t.Fatalf("Decide(candidate mode) failed: %v", err)
	}
	if dec.Route != HybridRouteCandidate || !dec.ShouldCallLLM {
		t.Fatalf("expected candidate route for LLM handoff, got %+v", dec)
	}
	if dec.Prediction.Intent == DefaultUnknownIntent || len(dec.Prediction.Candidates) == 0 {
		t.Fatalf("expected ranked candidates for LLM handoff, got %+v", dec.Prediction)
	}
}
