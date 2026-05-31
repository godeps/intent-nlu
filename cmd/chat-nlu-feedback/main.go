package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	chatnlu "github.com/godeps/intent-nlu"
)

type feedbackRecord struct {
	Text        string
	Language    string
	PredIntent  string
	Confidence  float64
	FinalIntent string
}

func main() {
	var (
		inputCSV      string
		defaultDir    string
		reviewDir     string
		lowConfidence float64
		unknownIntent string
	)

	flag.StringVar(&inputCSV, "input", "", "input feedback CSV path")
	flag.StringVar(&defaultDir, "default-dir", "./datasets/default", "default business dataset directory")
	flag.StringVar(&reviewDir, "review-dir", "./datasets/feedback/review", "review dataset output directory")
	flag.Float64Var(&lowConfidence, "low-confidence", 0.6, "confidence threshold for review samples")
	flag.StringVar(&unknownIntent, "unknown-intent", chatnlu.DefaultUnknownIntent, "unknown intent label")
	flag.Parse()

	if strings.TrimSpace(inputCSV) == "" {
		log.Fatal("-input is required")
	}
	records, err := loadFeedbackCSV(inputCSV)
	if err != nil {
		log.Fatalf("load input feedback csv failed: %v", err)
	}
	if len(records) == 0 {
		log.Fatal("no feedback rows found")
	}

	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		log.Fatalf("create default dir failed: %v", err)
	}
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		log.Fatalf("create review dir failed: %v", err)
	}

	aliases := chatnlu.DefaultIntentAliases()

	updates := map[string][]chatnlu.Sample{}
	reviews := map[string][]feedbackRecord{}
	for _, rec := range records {
		text := strings.TrimSpace(rec.Text)
		if text == "" {
			continue
		}
		lang := strings.ToLower(strings.TrimSpace(rec.Language))
		if lang == "" {
			lang = string(chatnlu.DetectLanguage(text))
		}
		if lang == "" {
			lang = "en"
		}

		finalIntent := chatnlu.NormalizeIntent(rec.FinalIntent, aliases)
		predIntent := chatnlu.NormalizeIntent(rec.PredIntent, aliases)
		if finalIntent != "" && finalIntent != unknownIntent {
			updates[lang] = append(updates[lang], chatnlu.Sample{Text: text, Intent: finalIntent})
			continue
		}
		if predIntent == "" {
			predIntent = unknownIntent
		}
		if predIntent == unknownIntent || rec.Confidence < lowConfidence {
			rec.Language = lang
			rec.PredIntent = predIntent
			reviews[lang] = append(reviews[lang], rec)
		}
	}

	updatedRows := 0
	for lang, samples := range updates {
		if len(samples) == 0 {
			continue
		}
		path := filepath.Join(defaultDir, lang+"_business.csv")
		existing, _ := chatnlu.LoadSamplesCSV(path)
		merged := mergeSamples(existing, samples)
		if err := chatnlu.SaveSamplesCSV(path, merged); err != nil {
			log.Fatalf("save merged dataset failed (%s): %v", path, err)
		}
		updatedRows += len(samples)
	}

	reviewRows := 0
	for lang, rows := range reviews {
		if len(rows) == 0 {
			continue
		}
		path := filepath.Join(reviewDir, lang+"_review.csv")
		if err := appendReviewRows(path, rows); err != nil {
			log.Fatalf("save review dataset failed (%s): %v", path, err)
		}
		reviewRows += len(rows)
	}

	fmt.Printf("feedback processed\n")
	fmt.Printf("input_rows=%d\n", len(records))
	fmt.Printf("updated_rows=%d\n", updatedRows)
	fmt.Printf("review_rows=%d\n", reviewRows)
}

func loadFeedbackCSV(path string) ([]feedbackRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	headers := mapHeaderIndex(rows[0])
	if _, ok := headers["text"]; !ok {
		return nil, errors.New("input csv requires text column")
	}
	result := make([]feedbackRecord, 0, len(rows)-1)
	for _, row := range rows[1:] {
		rec := feedbackRecord{}
		rec.Text = getColumn(row, headers, "text")
		rec.Language = getColumn(row, headers, "language")
		rec.PredIntent = getColumn(row, headers, "pred_intent", "intent", "predicted_intent")
		rec.FinalIntent = getColumn(row, headers, "final_intent", "human_intent", "label")
		confidenceText := getColumn(row, headers, "confidence", "score")
		if confidenceText != "" {
			if value, err := strconv.ParseFloat(confidenceText, 64); err == nil {
				rec.Confidence = value
			}
		}
		if strings.TrimSpace(rec.Text) == "" {
			continue
		}
		result = append(result, rec)
	}
	return result, nil
}

func mapHeaderIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, col := range header {
		m[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return m
}

func getColumn(row []string, index map[string]int, keys ...string) string {
	for _, key := range keys {
		if pos, ok := index[strings.ToLower(strings.TrimSpace(key))]; ok {
			if pos >= 0 && pos < len(row) {
				return strings.TrimSpace(row[pos])
			}
		}
	}
	return ""
}

func mergeSamples(existing []chatnlu.Sample, incoming []chatnlu.Sample) []chatnlu.Sample {
	seen := map[string]struct{}{}
	result := make([]chatnlu.Sample, 0, len(existing)+len(incoming))
	for _, item := range existing {
		text := strings.TrimSpace(item.Text)
		intent := strings.TrimSpace(item.Intent)
		if text == "" || intent == "" {
			continue
		}
		key := strings.ToLower(intent) + "\x00" + text
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, chatnlu.Sample{Text: text, Intent: intent})
	}
	for _, item := range incoming {
		text := strings.TrimSpace(item.Text)
		intent := strings.TrimSpace(item.Intent)
		if text == "" || intent == "" {
			continue
		}
		key := strings.ToLower(intent) + "\x00" + text
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, chatnlu.Sample{Text: text, Intent: intent})
	}
	return result
}

func appendReviewRows(path string, rows []feedbackRecord) error {
	existing := map[string]struct{}{}
	if f, err := os.Open(path); err == nil {
		r := csv.NewReader(f)
		r.FieldsPerRecord = -1
		all, _ := r.ReadAll()
		for _, row := range all[1:] {
			if len(row) < 3 {
				continue
			}
			key := row[0] + "\x00" + row[1] + "\x00" + row[2]
			existing[key] = struct{}{}
		}
		_ = f.Close()
	}

	createHeader := false
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		createHeader = true
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if createHeader {
		if err := w.Write([]string{"text", "language", "pred_intent", "confidence", "final_intent", "status"}); err != nil {
			return err
		}
	}
	for _, row := range rows {
		key := row.Text + "\x00" + row.Language + "\x00" + row.PredIntent
		if _, ok := existing[key]; ok {
			continue
		}
		existing[key] = struct{}{}
		record := []string{
			row.Text,
			row.Language,
			row.PredIntent,
			fmt.Sprintf("%.6f", row.Confidence),
			row.FinalIntent,
			"pending_review",
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
