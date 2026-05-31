package intentnlu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// BundleManifestFileName is the default file name of multilingual router bundle manifest.
	BundleManifestFileName = "manifest.json"
)

// BundleManifest describes one multilingual model bundle.
type BundleManifest struct {
	Version         string                        `json:"version"`
	CreatedAt       time.Time                     `json:"createdAt"`
	DefaultLanguage string                        `json:"defaultLanguage"`
	Corpus          SourceMetadata                `json:"corpus,omitempty"`
	TrainingParams  map[string]string             `json:"trainingParams,omitempty"`
	ModelSummary    map[string]BundleModelSummary `json:"modelSummary,omitempty"`
	Models          map[string]string             `json:"models"` // lang -> relative model directory
}

// BundleModelSummary stores high-level model metadata in bundle manifest.
type BundleModelSummary struct {
	Version             string  `json:"version,omitempty"`
	Language            string  `json:"language,omitempty"`
	TrainingSampleCount int     `json:"trainingSampleCount,omitempty"`
	TotalSampleCount    int     `json:"totalSampleCount,omitempty"`
	DefaultThreshold    float64 `json:"defaultThreshold,omitempty"`
	MacroF1             float64 `json:"macroF1,omitempty"`
}

// NewRouterFromBundle loads multilingual models from a bundle directory.
func NewRouterFromBundle(bundleDir string) (*Router, error) {
	bundleDir = strings.TrimSpace(bundleDir)
	if bundleDir == "" {
		return NewRouterFromEmbedded()
	}
	manifest, err := LoadBundleManifest(bundleDir)
	if err != nil {
		return nil, err
	}
	if len(manifest.Models) == 0 {
		return nil, fmt.Errorf("bundle has no models")
	}

	resolved := make(map[string]string, len(manifest.Models))
	for lang, relPath := range manifest.Models {
		lang = strings.TrimSpace(lang)
		relPath = strings.TrimSpace(relPath)
		if lang == "" || relPath == "" {
			continue
		}
		path := relPath
		if !filepath.IsAbs(path) {
			path = filepath.Join(bundleDir, path)
		}
		resolved[lang] = path
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("bundle has no valid model entries")
	}

	defaultLang := strings.TrimSpace(manifest.DefaultLanguage)
	if defaultLang == "" {
		langs := make([]string, 0, len(resolved))
		for lang := range resolved {
			langs = append(langs, lang)
		}
		sort.Strings(langs)
		defaultLang = langs[0]
	}
	return NewRouterFromDirs(resolved, defaultLang)
}

// LoadBundleManifest reads manifest.json from a bundle directory.
func LoadBundleManifest(bundleDir string) (BundleManifest, error) {
	bundleDir = strings.TrimSpace(bundleDir)
	if bundleDir == "" {
		return BundleManifest{}, fmt.Errorf("bundle dir is required")
	}
	path := filepath.Join(bundleDir, BundleManifestFileName)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return BundleManifest{}, fmt.Errorf("read bundle manifest: %w", err)
	}
	var manifest BundleManifest
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		return BundleManifest{}, fmt.Errorf("parse bundle manifest: %w", err)
	}
	if len(manifest.Models) == 0 {
		return BundleManifest{}, fmt.Errorf("bundle manifest has empty models")
	}
	return manifest, nil
}

// SaveBundleManifest writes bundle manifest into bundle directory.
func SaveBundleManifest(bundleDir string, manifest BundleManifest) error {
	if strings.TrimSpace(bundleDir) == "" {
		return fmt.Errorf("bundle dir is required")
	}
	if len(manifest.Models) == 0 {
		return fmt.Errorf("manifest models cannot be empty")
	}
	if manifest.CreatedAt.IsZero() {
		manifest.CreatedAt = time.Now().UTC()
	}
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("create bundle dir: %w", err)
	}
	bytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bundle manifest: %w", err)
	}
	path := filepath.Join(bundleDir, BundleManifestFileName)
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("write bundle manifest: %w", err)
	}
	return nil
}
