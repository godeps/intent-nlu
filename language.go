package chatnlu

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
	text = strings.TrimSpace(text)
	if text == "" {
		return LanguageEN
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
			return LanguageJA
		}
		if unicode.In(r, unicode.Hangul) {
			return LanguageKO
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			latin++
		}
	}

	if han > 0 && han*2 >= total {
		return LanguageZH
	}
	if latin > 0 {
		return LanguageEN
	}
	if han > 0 {
		return LanguageZH
	}
	return LanguageEN
}
