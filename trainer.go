package chatnlu

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jbrukh/bayesian"
)

// TrainedModel is an in-memory trained artifact.
type TrainedModel struct {
	classifier *bayesian.Classifier
	meta       ModelMeta
}

// Meta returns a copy of model metadata.
func (m *TrainedModel) Meta() ModelMeta {
	if m == nil {
		return ModelMeta{}
	}
	meta := m.meta
	meta.Classes = append([]string(nil), m.meta.Classes...)
	if m.meta.Thresholds != nil {
		meta.Thresholds = make(map[string]float64, len(m.meta.Thresholds))
		for k, v := range m.meta.Thresholds {
			meta.Thresholds[k] = v
		}
	}
	meta.Tokenizer.Stopwords = append([]string(nil), m.meta.Tokenizer.Stopwords...)
	meta.Tokenizer.CustomDicts = append([]string(nil), m.meta.Tokenizer.CustomDicts...)
	return meta
}

// Train trains a bayesian model from labeled samples.
func Train(samples []Sample, cfg TrainConfig) (*TrainedModel, error) {
	if len(samples) == 0 {
		return nil, errors.New("samples cannot be empty")
	}
	if strings.TrimSpace(cfg.UnknownIntent) == "" {
		cfg.UnknownIntent = DefaultUnknownIntent
	}
	if cfg.DefaultThreshold <= 0 {
		cfg.DefaultThreshold = 0.55
	}

	tok, err := NewTokenizer(cfg.Tokenizer)
	if err != nil {
		return nil, err
	}

	normalized := make([]Sample, 0, len(samples))
	intents := make(map[string]struct{})
	for _, s := range samples {
		intent := strings.TrimSpace(s.Intent)
		if intent == "" {
			continue
		}
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		normalized = append(normalized, Sample{Text: text, Intent: intent})
		intents[intent] = struct{}{}
	}
	if len(normalized) == 0 {
		return nil, errors.New("no valid labeled samples")
	}
	if len(intents) < 2 {
		return nil, errors.New("bayesian classifier requires at least two intents")
	}

	classNames := make([]string, 0, len(intents))
	for intent := range intents {
		classNames = append(classNames, intent)
	}
	sort.Strings(classNames)

	classes := make([]bayesian.Class, 0, len(classNames))
	for _, name := range classNames {
		classes = append(classes, bayesian.Class(name))
	}
	classifier := bayesian.NewClassifier(classes...)

	used := 0
	for _, s := range normalized {
		tokens := tok.Tokenize(s.Text)
		if len(tokens) == 0 {
			continue
		}
		classifier.Learn(tokens, bayesian.Class(s.Intent))
		used++
	}
	if used == 0 {
		return nil, errors.New("all samples became empty after tokenization")
	}

	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = time.Now().UTC().Format("20060102150405")
	}

	meta := ModelMeta{
		Version:             version,
		Language:            string(defaultLanguageForTokenizer(cfg.Tokenizer.Language)),
		UnknownIntent:       cfg.UnknownIntent,
		DefaultThreshold:    cfg.DefaultThreshold,
		Thresholds:          copyThresholds(cfg.Thresholds),
		Classes:             classNames,
		Tokenizer:           tok.Config(),
		TrainingSampleCount: used,
		CreatedAt:           time.Now().UTC(),
	}

	return &TrainedModel{
		classifier: classifier,
		meta:       meta,
	}, nil
}

func copyThresholds(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		trimmed := strings.TrimSpace(k)
		if trimmed == "" {
			continue
		}
		if v <= 0 {
			continue
		}
		out[trimmed] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ensureClasses(meta ModelMeta, classifier *bayesian.Classifier) error {
	if classifier == nil {
		return errors.New("classifier is nil")
	}
	if len(classifier.Classes) < 2 {
		return errors.New("classifier has fewer than 2 classes")
	}
	if len(meta.Classes) == 0 {
		meta.Classes = make([]string, 0, len(classifier.Classes))
		for _, c := range classifier.Classes {
			meta.Classes = append(meta.Classes, string(c))
		}
	}
	if len(meta.Classes) != len(classifier.Classes) {
		return fmt.Errorf("meta classes size mismatch: meta=%d classifier=%d", len(meta.Classes), len(classifier.Classes))
	}
	return nil
}
