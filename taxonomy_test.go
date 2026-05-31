package chatnlu

import "testing"

func TestDefaultTaxonomyDisabled(t *testing.T) {
	cfg := DefaultTaxonomyConfig()
	if cfg.Enabled {
		t.Fatalf("expected default taxonomy disabled")
	}
}

func TestNormalizeIntentAndThresholds(t *testing.T) {
	aliases := mergeIntentAliases(DefaultIntentAliases(), map[string]string{
		"holiday_query": "calendar_info",
		"weather_query": "weather_info",
	})

	if got := NormalizeIntent("Holiday_Query", aliases); got != "calendar_info" {
		t.Fatalf("expected calendar_info, got %s", got)
	}
	if got := NormalizeIntent("weather", aliases); got != "weather_info" {
		t.Fatalf("expected weather_info, got %s", got)
	}
	if got := NormalizeIntent("custom_intent", aliases); got != "custom_intent" {
		t.Fatalf("expected custom_intent passthrough, got %s", got)
	}

	thresholds := NormalizeThresholds(map[string]float64{
		"holiday_query": 0.71,
		"weather":       0.66,
		"custom_intent": 0.42,
		"invalid":       -1,
	}, aliases)
	if thresholds["calendar_info"] != 0.71 {
		t.Fatalf("expected calendar threshold 0.71, got %v", thresholds["calendar_info"])
	}
	if thresholds["weather_info"] != 0.66 {
		t.Fatalf("expected weather threshold 0.66, got %v", thresholds["weather_info"])
	}
	if thresholds["custom_intent"] != 0.42 {
		t.Fatalf("expected custom threshold 0.42, got %v", thresholds["custom_intent"])
	}
	if _, ok := thresholds["invalid"]; ok {
		t.Fatalf("invalid threshold should be dropped")
	}
}

func TestSkillRoutingAliases(t *testing.T) {
	aliases := DefaultIntentAliases()

	cases := []struct {
		input string
		want  string
	}{
		{"video_production", "creative_video"},
		{"video_editing", "creative_video"},
		{"film_production", "creative_video"},
		{"ad_production", "creative_video"},
		{"image_generation", "creative_image"},
		{"poster_design", "creative_image"},
		{"music_creation", "creative_audio"},
		{"audio_production", "creative_audio"},
		{"3d_modeling", "creative_3d"},
		{"video_analysis", "media_analysis"},
		{"image_analysis", "media_analysis"},
		{"creative_video", "creative_video"},
		{"creative_image", "creative_image"},
		{"creative_audio", "creative_audio"},
		{"creative_3d", "creative_3d"},
		{"media_analysis", "media_analysis"},
	}
	for _, tc := range cases {
		got := NormalizeIntent(tc.input, aliases)
		if got != tc.want {
			t.Errorf("NormalizeIntent(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}

	thresholds := NormalizeThresholds(map[string]float64{
		"video_production": 0.65,
		"image_generation": 0.70,
		"3d_modeling":      0.75,
	}, aliases)
	if thresholds["creative_video"] != 0.65 {
		t.Errorf("expected creative_video threshold 0.65, got %v", thresholds["creative_video"])
	}
	if thresholds["creative_image"] != 0.70 {
		t.Errorf("expected creative_image threshold 0.70, got %v", thresholds["creative_image"])
	}
	if thresholds["creative_3d"] != 0.75 {
		t.Errorf("expected creative_3d threshold 0.75, got %v", thresholds["creative_3d"])
	}
}

func TestCanonicalIntentsFromClasses(t *testing.T) {
	got := canonicalIntentsFromClasses([]string{
		"calendar_info",
		"weather_info",
		"calendar_info",
		"",
		"weather_info",
		"chitchat_general",
	})
	if len(got) != 3 {
		t.Fatalf("expected 3 canonical intents, got %d (%v)", len(got), got)
	}
}
