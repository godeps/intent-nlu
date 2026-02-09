package chatnlu

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-ego/gse"
)

var (
	punctReplacer = strings.NewReplacer(
		",", " ",
		".", " ",
		"?", " ",
		"!", " ",
		";", " ",
		":", " ",
		"，", " ",
		"。", " ",
		"？", " ",
		"！", " ",
		"；", " ",
		"：", " ",
		"（", " ",
		"）", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"【", " ",
		"】", " ",
		"{", " ",
		"}", " ",
		"<", " ",
		">", " ",
		"《", " ",
		"》", " ",
		"\"", " ",
		"'", " ",
		"`", " ",
		"、", " ",
	)
	spaceRe = regexp.MustCompile(`\s+`)
)

// Tokenizer wraps language-specific tokenization with normalization and filtering.
type Tokenizer struct {
	seg  *gse.Segmenter
	cfg  TokenizerConfig
	stop map[string]struct{}
}

// NewTokenizer creates a tokenizer from config.
func NewTokenizer(cfg TokenizerConfig) (*Tokenizer, error) {
	if cfg.MinTokenLen <= 0 {
		cfg.MinTokenLen = 1
	}
	cfg.Language = string(defaultLanguageForTokenizer(cfg.Language))

	var seg *gse.Segmenter
	if defaultLanguageForTokenizer(cfg.Language) == LanguageZH {
		instance := &gse.Segmenter{SkipLog: true}
		if err := instance.LoadDict(); err != nil {
			return nil, fmt.Errorf("load gse default dict: %w", err)
		}
		for _, path := range cfg.CustomDicts {
			trimmed := strings.TrimSpace(path)
			if trimmed == "" {
				continue
			}
			if err := instance.LoadDict(trimmed); err != nil {
				return nil, fmt.Errorf("load gse custom dict %s: %w", trimmed, err)
			}
		}
		seg = instance
	}

	stop := make(map[string]struct{}, len(cfg.Stopwords))
	for _, w := range cfg.Stopwords {
		norm := normalizeText(w, cfg)
		if norm == "" {
			continue
		}
		stop[norm] = struct{}{}
	}

	return &Tokenizer{seg: seg, cfg: cfg, stop: stop}, nil
}

// Config returns a copy of tokenizer config.
func (t *Tokenizer) Config() TokenizerConfig {
	if t == nil {
		return DefaultTokenizerConfig()
	}
	cfg := t.cfg
	cfg.Stopwords = append([]string(nil), t.cfg.Stopwords...)
	cfg.CustomDicts = append([]string(nil), t.cfg.CustomDicts...)
	return cfg
}

// Normalize exposes text normalization for testing and dataset preprocessing.
func (t *Tokenizer) Normalize(text string) string {
	if t == nil {
		return ""
	}
	return normalizeText(text, t.cfg)
}

// Tokenize tokenizes normalized text and filters invalid tokens.
func (t *Tokenizer) Tokenize(text string) []string {
	if t == nil {
		return nil
	}
	normalized := normalizeText(text, t.cfg)
	if normalized == "" {
		return nil
	}

	var raw []string
	lang := defaultLanguageForTokenizer(t.cfg.Language)
	if lang == LanguageZH && t.seg != nil {
		if t.cfg.SearchMode {
			raw = t.seg.CutSearch(normalized, t.cfg.HMM)
		} else {
			raw = t.seg.Cut(normalized, t.cfg.HMM)
		}
	} else {
		raw = basicTokenize(normalized, lang)
	}

	tokens := make([]string, 0, len(raw))
	for _, tok := range raw {
		tok = normalizeText(tok, t.cfg)
		if tok == "" {
			continue
		}
		if utf8.RuneCountInString(tok) < t.cfg.MinTokenLen {
			continue
		}
		if _, ok := t.stop[tok]; ok {
			continue
		}
		tokens = append(tokens, tok)
	}
	return tokens
}

func basicTokenize(text string, lang Language) []string {
	fields := strings.Fields(text)
	if len(fields) > 0 {
		return fields
	}
	if isCJKLanguage(lang) {
		runes := []rune(text)
		out := make([]string, 0, len(runes))
		for _, r := range runes {
			if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
				continue
			}
			out = append(out, string(r))
		}
		return out
	}
	return nil
}

func normalizeText(text string, cfg TokenizerConfig) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if cfg.Lowercase {
		text = strings.ToLower(text)
	}

	if cfg.StripPunct {
		text = punctReplacer.Replace(text)
		text = strings.Map(func(r rune) rune {
			if unicode.IsPunct(r) {
				return ' '
			}
			if unicode.IsControl(r) {
				return ' '
			}
			return r
		}, text)
	}

	if cfg.CollapseSpace {
		text = spaceRe.ReplaceAllString(text, " ")
	}
	return strings.TrimSpace(text)
}
