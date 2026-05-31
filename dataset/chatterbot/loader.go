package chatterbot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	chatnlu "github.com/godeps/intent-nlu"
	"gopkg.in/yaml.v3"
)

// LoaderOptions controls chatterbot corpus loading behavior.
type LoaderOptions struct {
	FileIntent     map[string]string
	CategoryIntent map[string]string
	DefaultIntent  string
	SkipUnmapped   bool
	IncludeReplies bool
}

// DefaultLoaderOptions returns a practical default loader config.
func DefaultLoaderOptions() LoaderOptions {
	return LoaderOptions{
		SkipUnmapped:   true,
		IncludeReplies: true,
	}
}

type corpusFile struct {
	Categories    []string `yaml:"categories"`
	Conversations []any    `yaml:"conversations"`
}

// LoadFileIntentMap loads a YAML map for file-name to intent mapping.
func LoadFileIntentMap(path string) (map[string]string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mapping := map[string]string{}
	if err := yaml.Unmarshal(bytes, &mapping); err != nil {
		return nil, fmt.Errorf("parse yaml map: %w", err)
	}
	for k, v := range mapping {
		trimmedK := strings.TrimSpace(k)
		trimmedV := strings.TrimSpace(v)
		if trimmedK == "" || trimmedV == "" {
			delete(mapping, k)
			continue
		}
		if trimmedK != k {
			delete(mapping, k)
			mapping[trimmedK] = trimmedV
		} else {
			mapping[k] = trimmedV
		}
	}
	return mapping, nil
}

// LoadSamples loads labeled samples from chatterbot Chinese corpus files.
func LoadSamples(root string, opts LoaderOptions) ([]chatnlu.Sample, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("root path is required")
	}
	if opts.FileIntent == nil {
		opts.FileIntent = map[string]string{}
	}
	if opts.CategoryIntent == nil {
		opts.CategoryIntent = map[string]string{}
	}

	files, err := collectYAMLFiles(root)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	unique := make(map[string]struct{})
	samples := make([]chatnlu.Sample, 0, 1024)
	for _, file := range files {
		parsed, err := parseCorpusFile(file)
		if err != nil {
			return nil, err
		}
		intent := resolveIntent(file, parsed.Categories, opts)
		if intent == "" && opts.SkipUnmapped {
			continue
		}
		if intent == "" {
			intent = strings.TrimSpace(opts.DefaultIntent)
		}
		if intent == "" {
			continue
		}

		for _, rawConvo := range parsed.Conversations {
			convo := normalizeConversation(rawConvo)
			if len(convo) == 0 {
				continue
			}
			limit := len(convo)
			if !opts.IncludeReplies {
				limit = 1
			}
			for i := 0; i < limit; i++ {
				text := strings.TrimSpace(convo[i])
				if text == "" {
					continue
				}
				key := intent + "\x00" + text
				if _, exists := unique[key]; exists {
					continue
				}
				unique[key] = struct{}{}
				samples = append(samples, chatnlu.Sample{Text: text, Intent: intent})
			}
		}
	}

	sort.SliceStable(samples, func(i, j int) bool {
		if samples[i].Intent == samples[j].Intent {
			return samples[i].Text < samples[j].Text
		}
		return samples[i].Intent < samples[j].Intent
	})
	return samples, nil
}

func collectYAMLFiles(root string) ([]string, error) {
	files := make([]string, 0, 128)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func parseCorpusFile(path string) (*corpusFile, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read corpus file %s: %w", path, err)
	}
	var parsed corpusFile
	if err := yaml.Unmarshal(bytes, &parsed); err != nil {
		return nil, fmt.Errorf("parse corpus file %s: %w", path, err)
	}
	return &parsed, nil
}

func resolveIntent(path string, categories []string, opts LoaderOptions) string {
	base := filepath.Base(path)
	if mapped := strings.TrimSpace(opts.FileIntent[base]); mapped != "" {
		return mapped
	}
	for _, cat := range categories {
		mapped := strings.TrimSpace(opts.CategoryIntent[strings.TrimSpace(cat)])
		if mapped != "" {
			return mapped
		}
	}
	return strings.TrimSpace(opts.DefaultIntent)
}

func normalizeConversation(raw any) []string {
	switch typed := raw.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		return []string{text}
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			out = append(out, item)
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			switch inner := item.(type) {
			case string:
				inner = strings.TrimSpace(inner)
				if inner != "" {
					out = append(out, inner)
				}
			default:
				text := strings.TrimSpace(fmt.Sprint(inner))
				if text != "" && text != "<nil>" {
					out = append(out, text)
				}
			}
		}
		return out
	default:
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text == "" || text == "<nil>" {
			return nil
		}
		return []string{text}
	}
}
