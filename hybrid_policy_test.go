package chatnlu

import (
	"context"
	"testing"
)

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
}
