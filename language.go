package intentnlu

import (
	"strings"
	"unicode"
)

// Language identifies tokenizer/model language.
type Language string

const (
	LanguageAuto Language = "auto"
	LanguageZH   Language = "zh"
	LanguageEN   Language = "en"
	LanguageJA   Language = "ja"
	LanguageKO   Language = "ko"
)

// LanguageDetection stores language detection result.
type LanguageDetection struct {
	Language    Language
	Confidence  float64
	Reason      string
	LetterCount int
	ShortText   bool
}

func normalizeLanguage(raw string) Language {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "", "auto":
		return LanguageAuto
	case "zh", "zh-cn", "zh-hans", "zh-tw", "zh-hant":
		return LanguageZH
	case "en", "en-us", "en-gb":
		return LanguageEN
	case "ja", "jp", "ja-jp":
		return LanguageJA
	case "ko", "ko-kr":
		return LanguageKO
	default:
		return Language(raw)
	}
}

func defaultLanguageForTokenizer(raw string) Language {
	lang := normalizeLanguage(raw)
	if lang == LanguageAuto {
		return LanguageZH
	}
	return lang
}

func isCJKLanguage(lang Language) bool {
	return lang == LanguageZH || lang == LanguageJA || lang == LanguageKO
}

// DetectLanguage returns a lightweight language guess for routing.
func DetectLanguage(text string) Language {
	return DetectLanguageDetailed(text).Language
}

// DetectLanguageDetailed returns language detection with confidence and reason.
func DetectLanguageDetailed(text string) LanguageDetection {
	text = strings.TrimSpace(text)
	if text == "" {
		return LanguageDetection{Language: LanguageEN, Confidence: 0.1, Reason: "empty_text", ShortText: true}
	}

	total := 0
	han := 0
	latin := 0
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsNumber(r) {
			continue
		}
		total++
		if unicode.In(r, unicode.Han) {
			han++
			continue
		}
		if unicode.In(r, unicode.Hiragana, unicode.Katakana) {
			return LanguageDetection{Language: LanguageJA, Confidence: 0.99, Reason: "kana_detected", LetterCount: total, ShortText: total <= 3}
		}
		if unicode.In(r, unicode.Hangul) {
			return LanguageDetection{Language: LanguageKO, Confidence: 0.99, Reason: "hangul_detected", LetterCount: total, ShortText: total <= 3}
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			latin++
		}
	}

	if total == 0 {
		return LanguageDetection{Language: LanguageEN, Confidence: 0.1, Reason: "no_letters", ShortText: true}
	}

	shortText := total <= 3
	if han > 0 && han*2 >= total {
		confidence := float64(han) / float64(total)
		if shortText {
			confidence *= 0.75
		}
		return LanguageDetection{
			Language:    LanguageZH,
			Confidence:  confidence,
			Reason:      "han_majority",
			LetterCount: total,
			ShortText:   shortText,
		}
	}
	if latin > 0 {
		confidence := float64(latin) / float64(total)
		if shortText {
			confidence *= 0.75
		}
		return LanguageDetection{
			Language:    LanguageEN,
			Confidence:  confidence,
			Reason:      "latin_majority",
			LetterCount: total,
			ShortText:   shortText,
		}
	}
	if han > 0 {
		confidence := float64(han) / float64(total)
		if shortText {
			confidence *= 0.75
		}
		return LanguageDetection{
			Language:    LanguageZH,
			Confidence:  confidence,
			Reason:      "han_partial",
			LetterCount: total,
			ShortText:   shortText,
		}
	}
	return LanguageDetection{
		Language:    LanguageEN,
		Confidence:  0.4,
		Reason:      "fallback_en",
		LetterCount: total,
		ShortText:   shortText,
	}
}
