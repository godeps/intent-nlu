package intentnlu

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const (
	HybridRouteRule      = "rule"
	HybridRouteNLU       = "nlu"
	HybridRouteCandidate = "candidate"
	HybridRouteFallback  = "fallback"
)

// DeterministicRule defines one pre-NLU deterministic route rule.
type DeterministicRule struct {
	ID          string   `json:"id"`
	Intent      string   `json:"intent"`
	Language    string   `json:"language,omitempty"`
	EqualsAny   []string `json:"equalsAny,omitempty"`
	PrefixAny   []string `json:"prefixAny,omitempty"`
	ContainsAny []string `json:"containsAny,omitempty"`
	Regex       string   `json:"regex,omitempty"`
	regexProg   *regexp.Regexp
}

// HybridDecision describes final routing decision.
type HybridDecision struct {
	Route         string     `json:"route"`
	Intent        string     `json:"intent,omitempty"`
	RuleID        string     `json:"ruleId,omitempty"`
	Prediction    Prediction `json:"prediction"`
	ShouldCallLLM bool       `json:"shouldCallLLM"`
}

// HybridPolicy combines deterministic rules + NLU + fallback. With
// PredictOptions.CandidateMode, NLU is used as a high-recall candidate
// generator and the final decision is left to the downstream LLM/tool planner.
type HybridPolicy struct {
	Rules         []DeterministicRule
	Router        *Router
	Engine        *Engine
	UnknownIntent string
}

// Prepare validates and compiles regex for hybrid rules.
func (p *HybridPolicy) Prepare() error {
	if p == nil {
		return fmt.Errorf("hybrid policy is nil")
	}
	if strings.TrimSpace(p.UnknownIntent) == "" {
		p.UnknownIntent = DefaultUnknownIntent
	}
	aliases := p.resolveAliases()
	for i := range p.Rules {
		p.Rules[i].Intent = strings.TrimSpace(p.Rules[i].Intent)
		if p.Rules[i].Intent == "" {
			continue
		}
		p.Rules[i].Intent = NormalizeIntent(p.Rules[i].Intent, aliases)
		if strings.TrimSpace(p.Rules[i].Regex) != "" {
			compiled, err := regexp.Compile(p.Rules[i].Regex)
			if err != nil {
				return fmt.Errorf("compile rule regex id=%s: %w", p.Rules[i].ID, err)
			}
			p.Rules[i].regexProg = compiled
		}
	}
	return nil
}

// Decide applies deterministic rules first, then NLU, then fallback.
func (p *HybridPolicy) Decide(ctx context.Context, text string, opts PredictOptions) (HybridDecision, error) {
	if p == nil {
		return HybridDecision{}, fmt.Errorf("hybrid policy is nil")
	}
	if strings.TrimSpace(p.UnknownIntent) == "" {
		p.UnknownIntent = DefaultUnknownIntent
	}

	detection := DetectLanguageDetailed(text)
	for _, rule := range p.Rules {
		if !ruleLanguageMatch(rule.Language, opts.LanguageHint, detection.Language) {
			continue
		}
		if !ruleTextMatch(rule, text) {
			continue
		}
		intent := rule.Intent
		if intent == "" {
			intent = p.UnknownIntent
		}
		return HybridDecision{
			Route:  HybridRouteRule,
			Intent: intent,
			RuleID: rule.ID,
			Prediction: Prediction{
				Intent:      intent,
				Language:    normalizePredictLanguage(opts.LanguageHint, detection.Language),
				Matched:     true,
				Reason:      "matched_deterministic_rule",
				UnknownUsed: intent == p.UnknownIntent,
			},
			ShouldCallLLM: false,
		}, nil
	}

	var (
		pred Prediction
		err  error
	)
	if p.Router != nil {
		pred, err = p.Router.Predict(ctx, text, opts)
	} else if p.Engine != nil {
		pred, err = p.Engine.Predict(ctx, text, opts)
	} else {
		return HybridDecision{}, fmt.Errorf("hybrid policy requires router or engine")
	}
	if err != nil {
		return HybridDecision{}, err
	}
	if opts.CandidateMode {
		return HybridDecision{
			Route:         HybridRouteCandidate,
			Intent:        pred.Intent,
			Prediction:    pred,
			ShouldCallLLM: true,
		}, nil
	}
	if pred.Matched && pred.Intent != p.UnknownIntent {
		return HybridDecision{
			Route:         HybridRouteNLU,
			Intent:        pred.Intent,
			Prediction:    pred,
			ShouldCallLLM: false,
		}, nil
	}
	return HybridDecision{
		Route:         HybridRouteFallback,
		Intent:        p.UnknownIntent,
		Prediction:    pred,
		ShouldCallLLM: true,
	}, nil
}

func (p *HybridPolicy) resolveAliases() map[string]string {
	if p.Router != nil {
		if aliases := p.Router.Meta().IntentAliases; len(aliases) > 0 {
			return aliases
		}
	}
	if p.Engine != nil {
		if aliases := p.Engine.Meta().IntentAliases; len(aliases) > 0 {
			return aliases
		}
	}
	return defaultIntentAliasesRef()
}

func normalizePredictLanguage(hint string, detected Language) string {
	lang := normalizeLanguage(hint)
	if lang != LanguageAuto {
		return string(lang)
	}
	if detected == LanguageAuto {
		return ""
	}
	return string(detected)
}

func ruleLanguageMatch(ruleLang string, hint string, detected Language) bool {
	ruleLang = strings.TrimSpace(ruleLang)
	if ruleLang == "" || strings.EqualFold(ruleLang, "any") {
		return true
	}
	ruleNorm := normalizeLanguage(ruleLang)
	if ruleNorm == LanguageAuto {
		return true
	}
	hintNorm := normalizeLanguage(hint)
	if hintNorm != LanguageAuto {
		return hintNorm == ruleNorm
	}
	if detected == LanguageAuto {
		return true
	}
	return detected == ruleNorm
}

func ruleTextMatch(rule DeterministicRule, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)

	for _, eq := range rule.EqualsAny {
		if strings.EqualFold(strings.TrimSpace(eq), text) {
			return true
		}
	}
	for _, prefix := range rule.PrefixAny {
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		if prefix != "" && strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	for _, part := range rule.ContainsAny {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" && strings.Contains(lower, part) {
			return true
		}
	}
	if rule.regexProg != nil && rule.regexProg.MatchString(text) {
		return true
	}
	return false
}
