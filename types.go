package intentnlu

import "time"

const (
	// DefaultUnknownIntent is returned when confidence is below threshold.
	DefaultUnknownIntent = "unknown"
	// ModelBinaryFile is the bayesian model artifact file name.
	ModelBinaryFile = "model.gob"
	// ModelMetaFile is the metadata artifact file name.
	ModelMetaFile = "meta.json"
)

// Sample is one supervised training sample.
type Sample struct {
	Text   string
	Intent string
}

// Candidate is one ranked prediction candidate.
type Candidate struct {
	Intent string  `json:"intent"`
	Score  float64 `json:"score"`
}

// Prediction is one inference result.
type Prediction struct {
	Intent      string      `json:"intent"`
	Language    string      `json:"language,omitempty"`
	Confidence  float64     `json:"confidence"`
	Strict      bool        `json:"strict"`
	Matched     bool        `json:"matched"`
	Reason      string      `json:"reason,omitempty"`
	Tokens      []string    `json:"tokens,omitempty"`
	Candidates  []Candidate `json:"candidates,omitempty"`
	ModelVer    string      `json:"modelVersion,omitempty"`
	UnknownUsed bool        `json:"unknownUsed"`
}

// TokenizerConfig defines tokenizer behavior.
type TokenizerConfig struct {
	Language      string   `json:"language,omitempty"`
	SearchMode    bool     `json:"searchMode"`
	HMM           bool     `json:"hmm"`
	Lowercase     bool     `json:"lowercase"`
	Stopwords     []string `json:"stopwords,omitempty"`
	CustomDicts   []string `json:"customDicts,omitempty"`
	MinTokenLen   int      `json:"minTokenLen"`
	StripPunct    bool     `json:"stripPunct"`
	CollapseSpace bool     `json:"collapseSpace"`
}

// DefaultTokenizerConfig returns a practical default tokenizer config.
func DefaultTokenizerConfig() TokenizerConfig {
	return TokenizerConfig{
		Language:      string(LanguageZH),
		SearchMode:    true,
		HMM:           true,
		Lowercase:     true,
		MinTokenLen:   1,
		StripPunct:    true,
		CollapseSpace: true,
	}
}

// DatasetSplitConfig controls deterministic train/val/test split.
type DatasetSplitConfig struct {
	Enabled    bool    `json:"enabled"`
	TrainRatio float64 `json:"trainRatio"`
	ValRatio   float64 `json:"valRatio"`
	TestRatio  float64 `json:"testRatio"`
	Seed       int64   `json:"seed"`
}

// DefaultDatasetSplitConfig returns default split config.
func DefaultDatasetSplitConfig() DatasetSplitConfig {
	return DatasetSplitConfig{
		Enabled:    false,
		TrainRatio: 0.8,
		ValRatio:   0.1,
		TestRatio:  0.1,
		Seed:       42,
	}
}

// TaxonomyConfig controls intent canonicalization.
type TaxonomyConfig struct {
	Enabled bool              `json:"enabled"`
	Aliases map[string]string `json:"aliases,omitempty"`
}

// DefaultTaxonomyConfig returns default taxonomy config.
func DefaultTaxonomyConfig() TaxonomyConfig {
	return TaxonomyConfig{
		Enabled: false,
		Aliases: defaultIntentAliasesRef(),
	}
}

// SourceMetadata describes training data source for reproducibility.
type SourceMetadata struct {
	Name     string            `json:"name,omitempty"`
	Version  string            `json:"version,omitempty"`
	Revision string            `json:"revision,omitempty"`
	RepoURL  string            `json:"repoUrl,omitempty"`
	Commit   string            `json:"commit,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

// TrainConfig controls training behavior.
type TrainConfig struct {
	Version                 string             `json:"version"`
	UnknownIntent           string             `json:"unknownIntent"`
	DefaultThreshold        float64            `json:"defaultThreshold"`
	Thresholds              map[string]float64 `json:"thresholds,omitempty"`
	Tokenizer               TokenizerConfig    `json:"tokenizer"`
	Split                   DatasetSplitConfig `json:"split"`
	AutoCalibrateThresholds bool               `json:"autoCalibrateThresholds"`
	Taxonomy                TaxonomyConfig     `json:"taxonomy"`
	Source                  SourceMetadata     `json:"source,omitempty"`
}

// DefaultTrainConfig returns a practical default training config.
func DefaultTrainConfig() TrainConfig {
	return TrainConfig{
		Version:                 time.Now().UTC().Format("20060102150405"),
		UnknownIntent:           DefaultUnknownIntent,
		DefaultThreshold:        0.55,
		Tokenizer:               DefaultTokenizerConfig(),
		Split:                   DefaultDatasetSplitConfig(),
		AutoCalibrateThresholds: false,
		Taxonomy:                DefaultTaxonomyConfig(),
	}
}

// PredictOptions controls prediction behavior.
type PredictOptions struct {
	TopK            int
	LanguageHint    string
	MinConfidence   float64 // if > 0, override model threshold
	IgnoreThreshold bool
}

// ClassMetrics describes one intent evaluation result.
type ClassMetrics struct {
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
	Support   int     `json:"support"`
	TP        int     `json:"tp"`
	FP        int     `json:"fp"`
	FN        int     `json:"fn"`
}

// EvalReport describes evaluation metrics for one split.
type EvalReport struct {
	Split       string                    `json:"split"`
	Samples     int                       `json:"samples"`
	Accuracy    float64                   `json:"accuracy"`
	MacroF1     float64                   `json:"macroF1"`
	MicroF1     float64                   `json:"microF1"`
	UnknownRate float64                   `json:"unknownRate"`
	PerIntent   map[string]ClassMetrics   `json:"perIntent"`
	Confusion   map[string]map[string]int `json:"confusion"`
}

// IntentDataSummary stores per-intent sample counts by split.
type IntentDataSummary struct {
	Total int `json:"total"`
	Train int `json:"train"`
	Val   int `json:"val"`
	Test  int `json:"test"`
}

// TrainingConfigSnapshot stores effective config used for training.
type TrainingConfigSnapshot struct {
	DefaultThreshold        float64            `json:"defaultThreshold"`
	Thresholds              map[string]float64 `json:"thresholds,omitempty"`
	Tokenizer               TokenizerConfig    `json:"tokenizer"`
	Split                   DatasetSplitConfig `json:"split"`
	AutoCalibrateThresholds bool               `json:"autoCalibrateThresholds"`
	TaxonomyEnabled         bool               `json:"taxonomyEnabled"`
}

// TrainingMetadata stores reproducibility and data summary info.
type TrainingMetadata struct {
	Seed                 int64                        `json:"seed"`
	TotalSampleCount     int                          `json:"totalSampleCount"`
	TrainSampleCount     int                          `json:"trainSampleCount"`
	ValSampleCount       int                          `json:"valSampleCount"`
	TestSampleCount      int                          `json:"testSampleCount"`
	Calibrated           bool                         `json:"calibrated"`
	CalibratedThresholds map[string]float64           `json:"calibratedThresholds,omitempty"`
	DataSummary          map[string]IntentDataSummary `json:"dataSummary,omitempty"`
	Config               TrainingConfigSnapshot       `json:"config"`
}

// ModelMeta stores model metadata and inference policy.
type ModelMeta struct {
	Version             string                `json:"version"`
	Language            string                `json:"language,omitempty"`
	UnknownIntent       string                `json:"unknownIntent"`
	DefaultThreshold    float64               `json:"defaultThreshold"`
	Thresholds          map[string]float64    `json:"thresholds,omitempty"`
	Classes             []string              `json:"classes"`
	CanonicalIntents    []string              `json:"canonicalIntents,omitempty"`
	IntentAliases       map[string]string     `json:"intentAliases,omitempty"`
	Tokenizer           TokenizerConfig       `json:"tokenizer"`
	TrainingSampleCount int                   `json:"trainingSampleCount"`
	CreatedAt           time.Time             `json:"createdAt"`
	Evaluation          map[string]EvalReport `json:"evaluation,omitempty"`
	Training            TrainingMetadata      `json:"training"`
	Source              SourceMetadata        `json:"source,omitempty"`
}
