package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	chatnlu "github.com/godeps/chat-nlu"
)

func main() {
	var (
		bundleDir      string
		modelsSpec     string
		defaultLang    string
		version        string
		corpusRepoURL  string
		corpusCommit   string
		trainingParams string
	)

	flag.StringVar(&bundleDir, "bundle-dir", "", "bundle output directory")
	flag.StringVar(&modelsSpec, "models", "", "model mapping: zh=./models/model-zh,en=./models/model-en")
	flag.StringVar(&defaultLang, "default-lang", "", "default router language")
	flag.StringVar(&version, "version", time.Now().UTC().Format("2006.01.02.bundle.150405"), "bundle version")
	flag.StringVar(&corpusRepoURL, "corpus-repo-url", "", "corpus repository URL")
	flag.StringVar(&corpusCommit, "corpus-commit", "", "corpus git commit")
	flag.StringVar(&trainingParams, "training-params", "", "training params map: key=value,key2=value2")
	flag.Parse()

	if strings.TrimSpace(bundleDir) == "" {
		log.Fatal("-bundle-dir is required")
	}
	if strings.TrimSpace(modelsSpec) == "" {
		log.Fatal("-models is required")
	}

	modelMap, err := parseModelsSpec(modelsSpec)
	if err != nil {
		log.Fatalf("parse model spec failed: %v", err)
	}
	if len(modelMap) == 0 {
		log.Fatal("no model mapping provided")
	}

	if strings.TrimSpace(defaultLang) == "" {
		langs := make([]string, 0, len(modelMap))
		for lang := range modelMap {
			langs = append(langs, lang)
		}
		sort.Strings(langs)
		defaultLang = langs[0]
	}

	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		log.Fatalf("create bundle dir failed: %v", err)
	}
	modelsOutDir := filepath.Join(bundleDir, "models")
	if err := os.MkdirAll(modelsOutDir, 0o755); err != nil {
		log.Fatalf("create bundle models dir failed: %v", err)
	}

	modelsRelative := make(map[string]string, len(modelMap))
	modelSummary := make(map[string]chatnlu.BundleModelSummary, len(modelMap))
	for lang, srcDir := range modelMap {
		lang = strings.TrimSpace(strings.ToLower(lang))
		if lang == "" {
			continue
		}
		dstDir := filepath.Join(modelsOutDir, lang)
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			log.Fatalf("create model dst dir failed: %v", err)
		}
		if err := copyFile(filepath.Join(srcDir, chatnlu.ModelBinaryFile), filepath.Join(dstDir, chatnlu.ModelBinaryFile)); err != nil {
			log.Fatalf("copy model binary failed (%s): %v", lang, err)
		}
		if err := copyFile(filepath.Join(srcDir, chatnlu.ModelMetaFile), filepath.Join(dstDir, chatnlu.ModelMetaFile)); err != nil {
			log.Fatalf("copy model meta failed (%s): %v", lang, err)
		}
		meta, err := loadMeta(filepath.Join(srcDir, chatnlu.ModelMetaFile))
		if err != nil {
			log.Fatalf("load model meta failed (%s): %v", lang, err)
		}
		macroF1 := 0.0
		if report, ok := meta.Evaluation["test"]; ok {
			macroF1 = report.MacroF1
		} else if report, ok := meta.Evaluation["val"]; ok {
			macroF1 = report.MacroF1
		}
		modelsRelative[lang] = filepath.ToSlash(filepath.Join("models", lang))
		modelSummary[lang] = chatnlu.BundleModelSummary{
			Version:             meta.Version,
			Language:            meta.Language,
			TrainingSampleCount: meta.TrainingSampleCount,
			TotalSampleCount:    meta.Training.TotalSampleCount,
			DefaultThreshold:    meta.DefaultThreshold,
			MacroF1:             macroF1,
		}
	}

	manifest := chatnlu.BundleManifest{
		Version:         strings.TrimSpace(version),
		CreatedAt:       time.Now().UTC(),
		DefaultLanguage: strings.TrimSpace(defaultLang),
		Corpus: chatnlu.SourceMetadata{
			RepoURL: strings.TrimSpace(corpusRepoURL),
			Commit:  strings.TrimSpace(corpusCommit),
		},
		TrainingParams: parseKVMap(trainingParams),
		ModelSummary:   modelSummary,
		Models:         modelsRelative,
	}
	if err := chatnlu.SaveBundleManifest(bundleDir, manifest); err != nil {
		log.Fatalf("save bundle manifest failed: %v", err)
	}
	fmt.Printf("bundle generated\n")
	fmt.Printf("bundle_dir=%s\n", bundleDir)
	fmt.Printf("models=%d\n", len(modelsRelative))
}

func parseModelsSpec(spec string) (map[string]string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("models spec is empty")
	}
	result := make(map[string]string)
	pairs := strings.Split(spec, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.IndexByte(pair, '=')
		if idx <= 0 || idx >= len(pair)-1 {
			return nil, fmt.Errorf("invalid models pair: %s", pair)
		}
		lang := strings.TrimSpace(pair[:idx])
		dir := strings.TrimSpace(pair[idx+1:])
		if lang == "" || dir == "" {
			return nil, fmt.Errorf("invalid models pair: %s", pair)
		}
		result[lang] = dir
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("models spec has no valid entries")
	}
	return result, nil
}

func parseKVMap(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		idx := strings.IndexByte(item, '=')
		if idx <= 0 || idx == len(item)-1 {
			continue
		}
		key := strings.TrimSpace(item[:idx])
		val := strings.TrimSpace(item[idx+1:])
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func copyFile(src, dst string) error {
	bytes, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, bytes, 0o644)
}

func loadMeta(path string) (chatnlu.ModelMeta, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return chatnlu.ModelMeta{}, err
	}
	var meta chatnlu.ModelMeta
	if err := json.Unmarshal(bytes, &meta); err != nil {
		return chatnlu.ModelMeta{}, err
	}
	return meta, nil
}
