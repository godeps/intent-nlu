package main

import (
	"os"
	"path/filepath"
	"testing"

	chatnlu "github.com/godeps/intent-nlu"
)

func TestLoadFeedbackCSVAndMergeSamples(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "feedback.csv")
	content := "text,language,pred_intent,confidence,final_intent\n" +
		"明天星期几,zh,calendar_info,0.93,calendar_info\n" +
		"上海天气,zh,weather_info,0.45,\n" +
		"hello,en,chitchat_greeting,0.88,chitchat_greeting\n"
	if err := os.WriteFile(input, []byte(content), 0o644); err != nil {
		t.Fatalf("write input csv failed: %v", err)
	}

	records, err := loadFeedbackCSV(input)
	if err != nil {
		t.Fatalf("loadFeedbackCSV failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	existing := []chatnlu.Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
	}
	incoming := []chatnlu.Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
		{Text: "hello", Intent: "chitchat_greeting"},
	}
	merged := mergeSamples(existing, incoming)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged samples, got %d", len(merged))
	}
}

func TestAppendReviewRowsDedup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zh_review.csv")
	rows := []feedbackRecord{
		{Text: "上海天气", Language: "zh", PredIntent: "weather_info", Confidence: 0.31},
		{Text: "上海天气", Language: "zh", PredIntent: "weather_info", Confidence: 0.31},
	}
	if err := appendReviewRows(path, rows); err != nil {
		t.Fatalf("appendReviewRows first write failed: %v", err)
	}
	if err := appendReviewRows(path, rows); err != nil {
		t.Fatalf("appendReviewRows second write failed: %v", err)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read review csv failed: %v", err)
	}
	lines := 0
	for _, c := range bytes {
		if c == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Fatalf("expected 2 csv lines (header + 1 row), got %d", lines)
	}
}
