package chatnlu

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Router routes inputs to language-specific intent engines.
type Router struct {
	mu          sync.RWMutex
	engines     map[Language]*Engine
	defaultLang Language
}

// NewRouter creates an empty router.
func NewRouter(defaultLang string) *Router {
	lang := normalizeLanguage(defaultLang)
	if lang == LanguageAuto {
		lang = LanguageEN
	}
	return &Router{
		engines:     map[Language]*Engine{},
		defaultLang: lang,
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

	lang := normalizeLanguage(opts.LanguageHint)
	if lang == LanguageAuto {
		lang = DetectLanguage(text)
	}

	engine, usedLang := r.pickEngine(lang)
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
