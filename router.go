package intentnlu

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// RouterOptions controls auto language routing behavior.
type RouterOptions struct {
	AutoDetectMinConfidence     float64
	ShortTextRuneLimit          int
	EnableCrossLanguageFallback bool
}

// DefaultRouterOptions returns default router behavior.
func DefaultRouterOptions() RouterOptions {
	return RouterOptions{
		AutoDetectMinConfidence:     0.72,
		ShortTextRuneLimit:          3,
		EnableCrossLanguageFallback: true,
	}
}

// Router routes inputs to language-specific intent engines.
type Router struct {
	mu          sync.RWMutex
	engines     map[Language]*Engine
	defaultLang Language
	options     RouterOptions
}

// NewRouter creates an empty router.
func NewRouter(defaultLang string) *Router {
	return NewRouterWithOptions(defaultLang, DefaultRouterOptions())
}

// NewRouterWithOptions creates an empty router with options.
func NewRouterWithOptions(defaultLang string, options RouterOptions) *Router {
	lang := normalizeLanguage(defaultLang)
	if lang == LanguageAuto {
		lang = LanguageEN
	}
	if options.AutoDetectMinConfidence <= 0 {
		options.AutoDetectMinConfidence = DefaultRouterOptions().AutoDetectMinConfidence
	}
	if options.ShortTextRuneLimit <= 0 {
		options.ShortTextRuneLimit = DefaultRouterOptions().ShortTextRuneLimit
	}
	return &Router{
		engines:     map[Language]*Engine{},
		defaultLang: lang,
		options:     options,
	}
}

// NewRouterFromDirs loads language models from directories.
func NewRouterFromDirs(modelByLanguage map[string]string, defaultLang string) (*Router, error) {
	r := NewRouter(defaultLang)
	for langText, dir := range modelByLanguage {
		lang := normalizeLanguage(langText)
		if lang == LanguageAuto {
			return nil, fmt.Errorf("language key cannot be auto")
		}
		if err := r.Load(langText, dir); err != nil {
			return nil, err
		}
		if r.defaultLang == LanguageAuto {
			r.defaultLang = lang
		}
	}
	if len(r.engines) == 0 {
		return nil, fmt.Errorf("no language model loaded")
	}
	if _, ok := r.engines[r.defaultLang]; !ok {
		for lang := range r.engines {
			r.defaultLang = lang
			break
		}
	}
	return r, nil
}

// Load loads one language model directory into router.
func (r *Router) Load(language string, modelDir string) error {
	if r == nil {
		return fmt.Errorf("router is nil")
	}
	lang := normalizeLanguage(language)
	if lang == LanguageAuto {
		return fmt.Errorf("language cannot be auto")
	}
	engine, err := NewEngineFromDir(modelDir)
	if err != nil {
		return err
	}

	actualLang := normalizeLanguage(engine.Language())
	if actualLang != LanguageAuto && actualLang != lang {
		lang = actualLang
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.engines == nil {
		r.engines = map[Language]*Engine{}
	}
	r.engines[lang] = engine
	if _, ok := r.engines[r.defaultLang]; !ok {
		r.defaultLang = lang
	}
	return nil
}

// Predict routes by language hint/detection and returns prediction.
func (r *Router) Predict(ctx context.Context, text string, opts PredictOptions) (Prediction, error) {
	if r == nil {
		return Prediction{}, fmt.Errorf("router is nil")
	}

	hint := normalizeLanguage(opts.LanguageHint)
	var selectedLang Language
	if hint != LanguageAuto {
		selectedLang = hint
	} else {
		detection := DetectLanguageDetailed(text)
		selectedLang = detection.Language
		if detection.Language == LanguageAuto {
			selectedLang = r.defaultLang
		}
		if r.options.EnableCrossLanguageFallback &&
			(detection.Confidence < r.options.AutoDetectMinConfidence || detection.ShortText || detection.LetterCount <= r.options.ShortTextRuneLimit) {
			if lang, ok := r.pickBestLanguageByRawScore(ctx, text, opts); ok {
				selectedLang = lang
			}
		}
	}

	engine, usedLang := r.pickEngine(selectedLang)
	if engine == nil {
		return Prediction{}, fmt.Errorf("no engine available")
	}

	pred, err := engine.Predict(ctx, text, opts)
	if err != nil {
		return pred, err
	}
	if strings.TrimSpace(pred.Language) == "" {
		pred.Language = string(usedLang)
	}
	return pred, nil
}

// Meta returns the default engine's model metadata, or empty if no engine is loaded.
func (r *Router) Meta() ModelMeta {
	if r == nil {
		return ModelMeta{}
	}
	engine, _ := r.pickEngine(r.defaultLang)
	if engine == nil {
		return ModelMeta{}
	}
	return engine.Meta()
}

// Languages returns loaded languages sorted alphabetically.
func (r *Router) Languages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.engines) == 0 {
		return nil
	}
	langs := make([]string, 0, len(r.engines))
	for lang := range r.engines {
		langs = append(langs, string(lang))
	}
	sort.Strings(langs)
	return langs
}

func (r *Router) pickEngine(lang Language) (*Engine, Language) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.engines) == 0 {
		return nil, LanguageAuto
	}
	if engine, ok := r.engines[lang]; ok {
		return engine, lang
	}
	if engine, ok := r.engines[r.defaultLang]; ok {
		return engine, r.defaultLang
	}
	for key, engine := range r.engines {
		return engine, key
	}
	return nil, LanguageAuto
}

func (r *Router) pickBestLanguageByRawScore(ctx context.Context, text string, opts PredictOptions) (Language, bool) {
	r.mu.RLock()
	if len(r.engines) == 0 {
		r.mu.RUnlock()
		return LanguageAuto, false
	}
	engines := make(map[Language]*Engine, len(r.engines))
	for lang, engine := range r.engines {
		engines[lang] = engine
	}
	defaultLang := r.defaultLang
	r.mu.RUnlock()

	rawOpts := opts
	rawOpts.MinConfidence = 0
	rawOpts.IgnoreThreshold = true

	if len(engines) == 1 {
		for lang, engine := range engines {
			pred, err := engine.Predict(ctx, text, rawOpts)
			if err != nil {
				return LanguageAuto, false
			}
			if pred.Confidence <= 0 {
				return LanguageAuto, false
			}
			return lang, true
		}
	}

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		bestLang  = LanguageAuto
		bestScore = -1.0
	)
	for lang, engine := range engines {
		wg.Add(1)
		go func(lang Language, engine *Engine) {
			defer wg.Done()
			pred, err := engine.Predict(ctx, text, rawOpts)
			if err != nil {
				return
			}
			score := pred.Confidence
			mu.Lock()
			if score > bestScore || (score == bestScore && lang == defaultLang) {
				bestScore = score
				bestLang = lang
			}
			mu.Unlock()
		}(lang, engine)
	}
	wg.Wait()

	if bestLang == LanguageAuto {
		return LanguageAuto, false
	}
	return bestLang, true
}
