package intentnlu

import (
	"strings"
	"testing"
)

func TestTokenizerTokenize_ZH(t *testing.T) {
	cfg := DefaultTokenizerConfig()
	cfg.Language = "zh"
	cfg.Stopwords = []string{"是"}
	tok, err := NewTokenizer(cfg)
	if err != nil {
		t.Fatalf("NewTokenizer failed: %v", err)
	}

	tokens := tok.Tokenize("明天是元宵节吗？")
	if len(tokens) == 0 {
		t.Fatalf("expected tokens, got empty")
	}
	for _, token := range tokens {
		if token == "是" {
			t.Fatalf("stopword not removed")
		}
		if strings.ContainsAny(token, "?？!！") {
			t.Fatalf("punctuation should be stripped, got token %q", token)
		}
	}
}

func TestTokenizerTokenize_EN(t *testing.T) {
	cfg := DefaultTokenizerConfig()
	cfg.Language = "en"
	cfg.Stopwords = []string{"is", "the"}
	tok, err := NewTokenizer(cfg)
	if err != nil {
		t.Fatalf("NewTokenizer failed: %v", err)
	}

	tokens := tok.Tokenize("What is the weather tomorrow?")
	if len(tokens) == 0 {
		t.Fatalf("expected tokens, got empty")
	}
	joined := strings.Join(tokens, " ")
	if strings.Contains(joined, " is ") || strings.HasPrefix(joined, "is ") || strings.HasSuffix(joined, " is") {
		t.Fatalf("stopword is not removed: %v", tokens)
	}
	if strings.Contains(joined, " the ") || strings.HasPrefix(joined, "the ") || strings.HasSuffix(joined, " the") {
		t.Fatalf("stopword the not removed: %v", tokens)
	}
}

func TestDetectLanguage(t *testing.T) {
	if got := DetectLanguage("明天星期几"); got != LanguageZH {
		t.Fatalf("expected zh, got %s", got)
	}
	if got := DetectLanguage("what is the weather tomorrow"); got != LanguageEN {
		t.Fatalf("expected en, got %s", got)
	}
}
