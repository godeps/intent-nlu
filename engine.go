package intentnlu

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/jbrukh/bayesian"
)

type engineState struct {
	modelDir   string
	classifier *bayesian.Classifier
	tokenizer  *Tokenizer
	meta       ModelMeta
}

// Engine provides concurrent-safe prediction and hot reload.
type Engine struct {
	current atomic.Value // *engineState
}

// NewEngineFromDir loads model artifacts and creates a prediction engine.
func NewEngineFromDir(modelDir string) (*Engine, error) {
	loaded, err := loadModelFromDir(modelDir)
	if err != nil {
		return nil, err
	}
	tok, err := NewTokenizer(loaded.meta.Tokenizer)
	if err != nil {
		return nil, err
	}
	state := &engineState{
		modelDir:   modelDir,
		classifier: loaded.classifier,
		tokenizer:  tok,
		meta:       loaded.meta,
	}
	eng := &Engine{}
	eng.current.Store(state)
	return eng, nil
}

// Reload atomically reloads model artifacts from the given directory.
func (e *Engine) Reload(modelDir string) error {
	if e == nil {
		return fmt.Errorf("engine is nil")
	}
	loaded, err := loadModelFromDir(modelDir)
	if err != nil {
		return err
	}
	tok, err := NewTokenizer(loaded.meta.Tokenizer)
	if err != nil {
		return err
	}
	e.current.Store(&engineState{
		modelDir:   modelDir,
		classifier: loaded.classifier,
		tokenizer:  tok,
		meta:       loaded.meta,
	})
	return nil
}

// Version returns current model version.
func (e *Engine) Version() string {
	state := e.state()
	if state == nil {
		return ""
	}
	return state.meta.Version
}

// Predict runs intent prediction.
func (e *Engine) Predict(_ context.Context, text string, opts PredictOptions) (Prediction, error) {
	state := e.state()
	if state == nil {
		return Prediction{}, fmt.Errorf("engine not initialized")
	}

	if opts.TopK <= 0 {
		if opts.CandidateMode {
			opts.TopK = 5
		} else {
			opts.TopK = 3
		}
	}
	tokens := state.tokenizer.Tokenize(text)
	if len(tokens) == 0 {
		return Prediction{
			Intent:      state.meta.UnknownIntent,
			Language:    state.meta.Language,
			Matched:     false,
			Reason:      "empty_tokens",
			ModelVer:    state.meta.Version,
			UnknownUsed: true,
		}, nil
	}

	bestClass, scores, strict := state.classifier.ClassifyProb(tokens)
	classes := state.classifier.Classes
	if len(scores) != len(classes) {
		return Prediction{}, fmt.Errorf("classifier returned invalid score length")
	}

	candidates := make([]Candidate, 0, len(scores))
	bestIntent := NormalizeIntent(strings.TrimSpace(string(bestClass)), state.meta.IntentAliases)
	bestScore := 0.0
	for i, score := range scores {
		intent := NormalizeIntent(string(classes[i]), state.meta.IntentAliases)
		candidates = append(candidates, Candidate{Intent: intent, Score: score})
		if intent == bestIntent {
			bestScore = score
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Intent < candidates[j].Intent
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > 0 && (bestIntent == "" || bestScore == 0) {
		bestIntent = candidates[0].Intent
		bestScore = candidates[0].Score
	}
	if opts.TopK < len(candidates) {
		candidates = candidates[:opts.TopK]
	}

	threshold := state.meta.DefaultThreshold
	if perIntent, ok := state.meta.Thresholds[bestIntent]; ok && perIntent > 0 {
		threshold = perIntent
	}
	if opts.MinConfidence > 0 {
		threshold = opts.MinConfidence
	}
	if opts.IgnoreThreshold || opts.CandidateMode {
		threshold = 0
	}

	if bestIntent == "" {
		return Prediction{
			Intent:      state.meta.UnknownIntent,
			Language:    state.meta.Language,
			Matched:     false,
			Reason:      "no_intent",
			Strict:      strict,
			Tokens:      tokens,
			Candidates:  candidates,
			ModelVer:    state.meta.Version,
			UnknownUsed: true,
		}, nil
	}

	if bestScore < threshold {
		return Prediction{
			Intent:      state.meta.UnknownIntent,
			Language:    state.meta.Language,
			Confidence:  bestScore,
			Strict:      strict,
			Matched:     false,
			Reason:      "below_threshold",
			Tokens:      tokens,
			Candidates:  candidates,
			ModelVer:    state.meta.Version,
			UnknownUsed: true,
		}, nil
	}

	return Prediction{
		Intent:      bestIntent,
		Language:    state.meta.Language,
		Confidence:  bestScore,
		Strict:      strict,
		Matched:     true,
		Tokens:      tokens,
		Candidates:  candidates,
		ModelVer:    state.meta.Version,
		UnknownUsed: false,
	}, nil
}

// Meta returns a deep copy of the current model metadata.
func (e *Engine) Meta() ModelMeta {
	state := e.state()
	if state == nil {
		return ModelMeta{}
	}
	meta := state.meta
	meta.Classes = append([]string(nil), state.meta.Classes...)
	meta.CanonicalIntents = append([]string(nil), state.meta.CanonicalIntents...)
	if state.meta.Thresholds != nil {
		meta.Thresholds = make(map[string]float64, len(state.meta.Thresholds))
		for k, v := range state.meta.Thresholds {
			meta.Thresholds[k] = v
		}
	}
	if state.meta.IntentAliases != nil {
		meta.IntentAliases = make(map[string]string, len(state.meta.IntentAliases))
		for k, v := range state.meta.IntentAliases {
			meta.IntentAliases[k] = v
		}
	}
	meta.Tokenizer.Stopwords = append([]string(nil), state.meta.Tokenizer.Stopwords...)
	meta.Tokenizer.CustomDicts = append([]string(nil), state.meta.Tokenizer.CustomDicts...)
	meta.Training = cloneTrainingMetadata(state.meta.Training)
	meta.Source = cloneSourceMetadata(state.meta.Source)
	if state.meta.Evaluation != nil {
		meta.Evaluation = cloneEvalReports(state.meta.Evaluation)
	}
	return meta
}

// Language returns current model language.
func (e *Engine) Language() string {
	state := e.state()
	if state == nil {
		return ""
	}
	return state.meta.Language
}

func (e *Engine) state() *engineState {
	if e == nil {
		return nil
	}
	cur := e.current.Load()
	if cur == nil {
		return nil
	}
	state, _ := cur.(*engineState)
	return state
}
