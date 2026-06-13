package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	intentnlu "github.com/godeps/intent-nlu"
)

func main() {
	var (
		modelDir      string
		modelsSpec    string
		bundleDir     string
		text          string
		langHint      string
		topK          int
		minConfidence float64
		candidateMode bool
	)

	flag.StringVar(&modelDir, "model", "", "single model directory containing model.gob and meta.json")
	flag.StringVar(&modelsSpec, "models", "", "multi-model mapping: zh=./model-zh,en=./model-en")
	flag.StringVar(&bundleDir, "bundle", "", "multilingual bundle directory containing manifest.json")
	flag.StringVar(&text, "text", "", "input text; if empty, read one line from stdin")
	flag.StringVar(&langHint, "lang", "auto", "language hint for routing (auto/zh/en/ja/ko)")
	flag.IntVar(&topK, "topk", 3, "number of candidates to output")
	flag.Float64Var(&minConfidence, "min-confidence", 0, "optional threshold override")
	flag.BoolVar(&candidateMode, "candidate-mode", false, "favor candidate recall for downstream LLM/tool-planner selection")
	flag.Parse()

	if strings.TrimSpace(text) == "" {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			log.Fatalf("read stdin failed: %v", err)
		}
		text = strings.TrimSpace(line)
	}
	if strings.TrimSpace(text) == "" {
		log.Fatal("input text is empty")
	}

	opts := intentnlu.PredictOptions{
		TopK:          topK,
		MinConfidence: minConfidence,
		LanguageHint:  langHint,
		CandidateMode: candidateMode,
	}

	var (
		pred intentnlu.Prediction
		err  error
	)
	if strings.TrimSpace(bundleDir) != "" {
		router, routerErr := intentnlu.NewRouterFromBundle(bundleDir)
		if routerErr != nil {
			log.Fatalf("load router bundle failed: %v", routerErr)
		}
		pred, err = router.Predict(context.Background(), text, opts)
	} else if strings.TrimSpace(modelsSpec) != "" {
		models, parseErr := parseModelsSpec(modelsSpec)
		if parseErr != nil {
			log.Fatalf("parse models failed: %v", parseErr)
		}
		router, routerErr := intentnlu.NewRouterFromDirs(models, langHint)
		if routerErr != nil {
			log.Fatalf("load router models failed: %v", routerErr)
		}
		pred, err = router.Predict(context.Background(), text, opts)
	} else if strings.TrimSpace(modelDir) != "" {
		engine, loadErr := intentnlu.NewEngineFromDir(modelDir)
		if loadErr != nil {
			log.Fatalf("load model failed: %v", loadErr)
		}
		pred, err = engine.Predict(context.Background(), text, opts)
	} else {
		router, routerErr := intentnlu.NewRouterFromEmbedded()
		if routerErr != nil {
			log.Fatalf("load embedded default bundle failed: %v", routerErr)
		}
		pred, err = router.Predict(context.Background(), text, opts)
	}
	if err != nil {
		log.Fatalf("predict failed: %v", err)
	}

	bytes, err := json.MarshalIndent(pred, "", "  ")
	if err != nil {
		log.Fatalf("marshal json failed: %v", err)
	}
	fmt.Println(string(bytes))
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
