package chatnlu

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

// TrainConfig controls training behavior.
type TrainConfig struct {
	Version          string             `json:"version"`
	UnknownIntent    string             `json:"unknownIntent"`
	DefaultThreshold float64            `json:"defaultThreshold"`
	Thresholds       map[string]float64 `json:"thresholds,omitempty"`
	Tokenizer        TokenizerConfig    `json:"tokenizer"`
}

// DefaultTrainConfig returns a practical default training config.
func DefaultTrainConfig() TrainConfig {
	return TrainConfig{
		Version:          time.Now().UTC().Format("20060102150405"),
		UnknownIntent:    DefaultUnknownIntent,
		DefaultThreshold: 0.55,
		Tokenizer:        DefaultTokenizerConfig(),
	}
}

// PredictOptions controls prediction behavior.
type PredictOptions struct {
	TopK          int
	LanguageHint  string
	MinConfidence float64 // if > 0, override model threshold
}

// ModelMeta stores model metadata and inference policy.
type ModelMeta struct {
	Version             string             `json:"version"`
	Language            string             `json:"language,omitempty"`
	UnknownIntent       string             `json:"unknownIntent"`
	DefaultThreshold    float64            `json:"defaultThreshold"`
	Thresholds          map[string]float64 `json:"thresholds,omitempty"`
	Classes             []string           `json:"classes"`
	Tokenizer           TokenizerConfig    `json:"tokenizer"`
	TrainingSampleCount int                `json:"trainingSampleCount"`
	CreatedAt           time.Time          `json:"createdAt"`
}
