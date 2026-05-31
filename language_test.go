package intentnlu

import "testing"

func TestDetectLanguageDetailed(t *testing.T) {
	zh := DetectLanguageDetailed("明天星期几")
	if zh.Language != LanguageZH {
		t.Fatalf("expected zh, got %+v", zh)
	}
	if zh.Confidence <= 0 {
		t.Fatalf("expected positive confidence, got %+v", zh)
	}

	en := DetectLanguageDetailed("weather tomorrow")
	if en.Language != LanguageEN {
		t.Fatalf("expected en, got %+v", en)
	}

	ja := DetectLanguageDetailed("こんにちは")
	if ja.Language != LanguageJA {
		t.Fatalf("expected ja, got %+v", ja)
	}

	ko := DetectLanguageDetailed("안녕하세요")
	if ko.Language != LanguageKO {
		t.Fatalf("expected ko, got %+v", ko)
	}
}
