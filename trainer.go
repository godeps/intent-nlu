package intentnlu

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
	meta.CanonicalIntents = append([]string(nil), m.meta.CanonicalIntents...)
	if m.meta.Thresholds != nil {
		meta.Thresholds = make(map[string]float64, len(m.meta.Thresholds))
		for k, v := range m.meta.Thresholds {
			meta.Thresholds[k] = v
		}
	}
	if m.meta.IntentAliases != nil {
		meta.IntentAliases = make(map[string]string, len(m.meta.IntentAliases))
		for k, v := range m.meta.IntentAliases {
			meta.IntentAliases[k] = v
		}
	}
	if m.meta.Evaluation != nil {
		meta.Evaluation = cloneEvalReports(m.meta.Evaluation)
	}
	meta.Tokenizer.Stopwords = append([]string(nil), m.meta.Tokenizer.Stopwords...)
	meta.Tokenizer.CustomDicts = append([]string(nil), m.meta.Tokenizer.CustomDicts...)
	meta.Training = cloneTrainingMetadata(m.meta.Training)
	meta.Source = cloneSourceMetadata(m.meta.Source)
	return meta
}

// Train trains a bayesian model from labeled samples and produces metadata.
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
	if cfg.Split.Seed == 0 {
		cfg.Split.Seed = 42
	}

	var aliases map[string]string
	if cfg.Taxonomy.Enabled {
		aliases = mergeIntentAliases(DefaultIntentAliases(), cfg.Taxonomy.Aliases)
	}

	tok, err := NewTokenizer(cfg.Tokenizer)
	if err != nil {
		return nil, err
	}

	processed, err := preprocessSamples(samples, tok, aliases, cfg.UnknownIntent)
	if err != nil {
		return nil, err
	}
	splits := splitTokenizedSamples(processed, cfg.Split)

	classifier, classes, err := trainClassifier(splits.Train)
	if err != nil && cfg.Split.Enabled {
		// Fallback to full dataset if split produced insufficient class coverage.
		splits = splitResult{Train: append([]tokenizedSample(nil), processed...)}
		classifier, classes, err = trainClassifier(splits.Train)
	}
	if err != nil {
		return nil, err
	}

	thresholds := NormalizeThresholds(copyThresholds(cfg.Thresholds), aliases)
	if thresholds == nil {
		thresholds = map[string]float64{}
	}

	calibrated := false
	if cfg.AutoCalibrateThresholds && len(splits.Val) > 0 {
		autoThresholds := calibrateThresholds(classifier, splits.Val, classes, cfg.DefaultThreshold)
		for intent, threshold := range autoThresholds {
			thresholds[intent] = threshold
		}
		// manual thresholds override auto-calibration
		manualThresholds := NormalizeThresholds(copyThresholds(cfg.Thresholds), aliases)
		for intent, threshold := range manualThresholds {
			thresholds[intent] = threshold
		}
		calibrated = true
	}

	evaluation := map[string]EvalReport{}
	evaluation["train"] = evaluateClassifier(classifier, splits.Train, classes, cfg.UnknownIntent, cfg.DefaultThreshold, thresholds, "train")
	if len(splits.Val) > 0 {
		evaluation["val"] = evaluateClassifier(classifier, splits.Val, classes, cfg.UnknownIntent, cfg.DefaultThreshold, thresholds, "val")
	}
	if len(splits.Test) > 0 {
		evaluation["test"] = evaluateClassifier(classifier, splits.Test, classes, cfg.UnknownIntent, cfg.DefaultThreshold, thresholds, "test")
	}

	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = time.Now().UTC().Format("20060102150405")
	}

	dataSummary := buildDataSummary(splits)
	canonicalIntents := canonicalIntentsFromClasses(classes)
	sort.Strings(canonicalIntents)

	meta := ModelMeta{
		Version:             version,
		Language:            string(defaultLanguageForTokenizer(cfg.Tokenizer.Language)),
		UnknownIntent:       cfg.UnknownIntent,
		DefaultThreshold:    cfg.DefaultThreshold,
		Thresholds:          thresholds,
		Classes:             append([]string(nil), classes...),
		CanonicalIntents:    canonicalIntents,
		IntentAliases:       aliases,
		Tokenizer:           tok.Config(),
		TrainingSampleCount: len(splits.Train),
		CreatedAt:           time.Now().UTC(),
		Evaluation:          evaluation,
		Training: TrainingMetadata{
			Seed:                 cfg.Split.Seed,
			TotalSampleCount:     len(processed),
			TrainSampleCount:     len(splits.Train),
			ValSampleCount:       len(splits.Val),
			TestSampleCount:      len(splits.Test),
			Calibrated:           calibrated,
			CalibratedThresholds: thresholds,
			DataSummary:          dataSummary,
			Config: TrainingConfigSnapshot{
				DefaultThreshold:        cfg.DefaultThreshold,
				Thresholds:              copyThresholds(cfg.Thresholds),
				Tokenizer:               tok.Config(),
				Split:                   cfg.Split,
				AutoCalibrateThresholds: cfg.AutoCalibrateThresholds,
				TaxonomyEnabled:         cfg.Taxonomy.Enabled,
			},
		},
		Source: cloneSourceMetadata(cfg.Source),
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

func buildDataSummary(splits splitResult) map[string]IntentDataSummary {
	result := map[string]IntentDataSummary{}
	for _, sample := range splits.Train {
		item := result[sample.Intent]
		item.Total++
		item.Train++
		result[sample.Intent] = item
	}
	for _, sample := range splits.Val {
		item := result[sample.Intent]
		item.Total++
		item.Val++
		result[sample.Intent] = item
	}
	for _, sample := range splits.Test {
		item := result[sample.Intent]
		item.Total++
		item.Test++
		result[sample.Intent] = item
	}
	return result
}

func cloneEvalReports(in map[string]EvalReport) map[string]EvalReport {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]EvalReport, len(in))
	for split, report := range in {
		copied := report
		if report.PerIntent != nil {
			copied.PerIntent = make(map[string]ClassMetrics, len(report.PerIntent))
			for key, value := range report.PerIntent {
				copied.PerIntent[key] = value
			}
		}
		if report.Confusion != nil {
			copied.Confusion = make(map[string]map[string]int, len(report.Confusion))
			for actual, row := range report.Confusion {
				copiedRow := make(map[string]int, len(row))
				for predicted, value := range row {
					copiedRow[predicted] = value
				}
				copied.Confusion[actual] = copiedRow
			}
		}
		out[split] = copied
	}
	return out
}

func cloneTrainingMetadata(in TrainingMetadata) TrainingMetadata {
	out := in
	if in.CalibratedThresholds != nil {
		out.CalibratedThresholds = make(map[string]float64, len(in.CalibratedThresholds))
		for key, value := range in.CalibratedThresholds {
			out.CalibratedThresholds[key] = value
		}
	}
	if in.DataSummary != nil {
		out.DataSummary = make(map[string]IntentDataSummary, len(in.DataSummary))
		for key, value := range in.DataSummary {
			out.DataSummary[key] = value
		}
	}
	if in.Config.Thresholds != nil {
		out.Config.Thresholds = make(map[string]float64, len(in.Config.Thresholds))
		for key, value := range in.Config.Thresholds {
			out.Config.Thresholds[key] = value
		}
	}
	out.Config.Tokenizer.Stopwords = append([]string(nil), in.Config.Tokenizer.Stopwords...)
	out.Config.Tokenizer.CustomDicts = append([]string(nil), in.Config.Tokenizer.CustomDicts...)
	return out
}

func cloneSourceMetadata(in SourceMetadata) SourceMetadata {
	out := in
	if in.Extra != nil {
		out.Extra = make(map[string]string, len(in.Extra))
		for key, value := range in.Extra {
			out.Extra[key] = value
		}
	}
	return out
}
