package intentnlu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbrukh/bayesian"
)

// SaveDir persists the trained model into a directory.
func (m *TrainedModel) SaveDir(dir string) error {
	if m == nil || m.classifier == nil {
		return fmt.Errorf("trained model is nil")
	}
	if err := ensureClasses(m.meta, m.classifier); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create model dir: %w", err)
	}
	modelPath := filepath.Join(dir, ModelBinaryFile)
	if err := m.classifier.WriteToFile(modelPath); err != nil {
		return fmt.Errorf("write model binary: %w", err)
	}

	meta := m.Meta()
	if strings.TrimSpace(meta.UnknownIntent) == "" {
		meta.UnknownIntent = DefaultUnknownIntent
	}
	if meta.DefaultThreshold <= 0 {
		meta.DefaultThreshold = 0.55
	}
	metaPath := filepath.Join(dir, ModelMetaFile)
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal model meta: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		return fmt.Errorf("write model meta: %w", err)
	}
	return nil
}

type loadedModel struct {
	classifier *bayesian.Classifier
	meta       ModelMeta
}

func loadModelFromDir(dir string) (*loadedModel, error) {
	modelPath := filepath.Join(dir, ModelBinaryFile)
	classifier, err := bayesian.NewClassifierFromFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load model binary: %w", err)
	}

	metaPath := filepath.Join(dir, ModelMetaFile)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read model meta: %w", err)
	}
	var meta ModelMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal model meta: %w", err)
	}
	if strings.TrimSpace(meta.UnknownIntent) == "" {
		meta.UnknownIntent = DefaultUnknownIntent
	}
	if meta.DefaultThreshold <= 0 {
		meta.DefaultThreshold = 0.55
	}
	if meta.Tokenizer.MinTokenLen <= 0 {
		meta.Tokenizer.MinTokenLen = 1
	}
	meta.Tokenizer.Language = string(defaultLanguageForTokenizer(meta.Tokenizer.Language))
	if strings.TrimSpace(meta.Language) == "" {
		meta.Language = meta.Tokenizer.Language
	}
	taxonomyEnabled := meta.Training.Config.TaxonomyEnabled || len(meta.IntentAliases) > 0
	if taxonomyEnabled {
		meta.IntentAliases = mergeIntentAliases(DefaultIntentAliases(), meta.IntentAliases)
		meta.Thresholds = NormalizeThresholds(meta.Thresholds, meta.IntentAliases)
	} else {
		meta.IntentAliases = nil
		meta.Thresholds = copyThresholds(meta.Thresholds)
	}
	if err := ensureClasses(meta, classifier); err != nil {
		return nil, err
	}

	if len(meta.Classes) == 0 {
		meta.Classes = make([]string, 0, len(classifier.Classes))
		for _, c := range classifier.Classes {
			meta.Classes = append(meta.Classes, string(c))
		}
	}
	for i := range meta.Classes {
		if taxonomyEnabled {
			meta.Classes[i] = NormalizeIntent(meta.Classes[i], meta.IntentAliases)
			continue
		}
		meta.Classes[i] = strings.TrimSpace(meta.Classes[i])
	}
	meta.CanonicalIntents = canonicalIntentsFromClasses(meta.Classes)
	return &loadedModel{classifier: classifier, meta: meta}, nil
}
