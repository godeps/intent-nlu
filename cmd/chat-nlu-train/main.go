package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	chatnlu "github.com/godeps/chat-nlu"
	"github.com/godeps/chat-nlu/dataset/chatterbot"
)

func main() {
	var (
		corpusRoot      string
		fileMapPath     string
		categoryMapPath string
		defaultIntent   string
		skipUnmapped    bool
		includeReplies  bool
		extraCSV        string
		dumpSamplesCSV  string
		outDir          string
		version         string
		unknownIntent   string
		defaultTh       float64
		thresholdsStr   string
		language        string
		searchMode      bool
		hmmMode         bool
		lowercase       bool
		stripPunct      bool
		stopwordsStr    string
		minTokenLen     int
	)

	flag.StringVar(&corpusRoot, "corpus-root", "", "chatterbot chinese corpus root directory")
	flag.StringVar(&fileMapPath, "file-map", "", "yaml file: filename.yml -> intent")
	flag.StringVar(&categoryMapPath, "category-map", "", "yaml file: category -> intent")
	flag.StringVar(&defaultIntent, "default-intent", "", "default intent for unmapped files")
	flag.BoolVar(&skipUnmapped, "skip-unmapped", true, "skip files without resolved intent")
	flag.BoolVar(&includeReplies, "include-replies", true, "include all utterances in conversations")
	flag.StringVar(&extraCSV, "extra-csv", "", "optional extra csv samples (text,intent)")
	flag.StringVar(&dumpSamplesCSV, "dump-samples", "", "optional path to dump the effective training samples (csv)")
	flag.StringVar(&outDir, "out", "./model", "output model directory")
	flag.StringVar(&version, "version", "", "model version")
	flag.StringVar(&unknownIntent, "unknown-intent", chatnlu.DefaultUnknownIntent, "unknown intent label")
	flag.Float64Var(&defaultTh, "threshold", 0.55, "default intent confidence threshold")
	flag.StringVar(&thresholdsStr, "thresholds", "", "per intent thresholds: intent=0.6,intent2=0.7")
	flag.StringVar(&language, "lang", "zh", "training language (zh/en/ja/ko)")
	flag.BoolVar(&searchMode, "search-mode", true, "use gse search mode")
	flag.BoolVar(&hmmMode, "hmm", true, "use gse hmm mode")
	flag.BoolVar(&lowercase, "lowercase", true, "lowercase input before tokenization")
	flag.BoolVar(&stripPunct, "strip-punct", true, "strip punctuation before tokenization")
	flag.StringVar(&stopwordsStr, "stopwords", "", "comma-separated stopwords")
	flag.IntVar(&minTokenLen, "min-token-len", 1, "minimum token rune length")
	flag.Parse()

	if strings.TrimSpace(corpusRoot) == "" && strings.TrimSpace(extraCSV) == "" {
		log.Fatal("at least one data source is required: -corpus-root or -extra-csv")
	}

	samples := make([]chatnlu.Sample, 0, 4096)
	if strings.TrimSpace(corpusRoot) != "" {
		fileIntent, err := loadMapFile(fileMapPath)
		if err != nil {
			log.Fatalf("load file map failed: %v", err)
		}
		categoryIntent, err := loadMapFile(categoryMapPath)
		if err != nil {
			log.Fatalf("load category map failed: %v", err)
		}
		loaded, err := chatterbot.LoadSamples(corpusRoot, chatterbot.LoaderOptions{
			FileIntent:     fileIntent,
			CategoryIntent: categoryIntent,
			DefaultIntent:  defaultIntent,
			SkipUnmapped:   skipUnmapped,
			IncludeReplies: includeReplies,
		})
		if err != nil {
			log.Fatalf("load chatterbot samples failed: %v", err)
		}
		samples = append(samples, loaded...)
	}

	if strings.TrimSpace(extraCSV) != "" {
		loaded, err := chatnlu.LoadSamplesCSV(extraCSV)
		if err != nil {
			log.Fatalf("load extra csv failed: %v", err)
		}
		samples = append(samples, loaded...)
	}
	if strings.TrimSpace(dumpSamplesCSV) != "" {
		dir := filepath.Dir(dumpSamplesCSV)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				log.Fatalf("create dump samples dir failed: %v", err)
			}
		}
		if err := chatnlu.SaveSamplesCSV(dumpSamplesCSV, samples); err != nil {
			log.Fatalf("dump training samples failed: %v", err)
		}
	}

	thresholds, err := parseThresholds(thresholdsStr)
	if err != nil {
		log.Fatalf("parse thresholds failed: %v", err)
	}
	cfg := chatnlu.DefaultTrainConfig()
	if strings.TrimSpace(version) != "" {
		cfg.Version = strings.TrimSpace(version)
	}
	cfg.UnknownIntent = strings.TrimSpace(unknownIntent)
	cfg.DefaultThreshold = defaultTh
	cfg.Thresholds = thresholds
	cfg.Tokenizer.Language = strings.TrimSpace(language)
	cfg.Tokenizer.SearchMode = searchMode
	cfg.Tokenizer.HMM = hmmMode
	cfg.Tokenizer.Lowercase = lowercase
	cfg.Tokenizer.StripPunct = stripPunct
	cfg.Tokenizer.MinTokenLen = minTokenLen
	cfg.Tokenizer.Stopwords = splitCSV(stopwordsStr)

	model, err := chatnlu.Train(samples, cfg)
	if err != nil {
		log.Fatalf("train failed: %v", err)
	}
	if err := model.SaveDir(outDir); err != nil {
		log.Fatalf("save model failed: %v", err)
	}

	meta := model.Meta()
	fmt.Printf("train succeeded\n")
	fmt.Printf("output=%s\n", outDir)
	fmt.Printf("version=%s\n", meta.Version)
	fmt.Printf("samples=%d\n", meta.TrainingSampleCount)
	fmt.Printf("classes=%d\n", len(meta.Classes))
	fmt.Printf("class_list=%s\n", strings.Join(sortedCopy(meta.Classes), ","))
}

func loadMapFile(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return map[string]string{}, nil
	}
	return chatterbot.LoadFileIntentMap(path)
}

func splitCSV(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	parts := strings.Split(text, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func parseThresholds(raw string) (map[string]float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	out := make(map[string]float64)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.IndexByte(pair, '=')
		if idx <= 0 || idx == len(pair)-1 {
			return nil, fmt.Errorf("invalid pair: %s", pair)
		}
		key := strings.TrimSpace(pair[:idx])
		valText := strings.TrimSpace(pair[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("empty intent in pair: %s", pair)
		}
		value, err := strconv.ParseFloat(valText, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid threshold for %s: %w", key, err)
		}
		if value <= 0 {
			return nil, fmt.Errorf("threshold must be > 0 for %s", key)
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
