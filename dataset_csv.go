package intentnlu

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoadSamplesCSV loads labeled samples from CSV.
// Expected columns: text,intent (header row optional).
func LoadSamplesCSV(path string) ([]Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	var samples []Sample
	line := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv line %d: %w", line+1, err)
		}
		line++
		if len(rec) < 2 {
			continue
		}
		text := strings.TrimSpace(rec[0])
		intent := strings.TrimSpace(rec[1])
		if text == "" || intent == "" {
			continue
		}
		if line == 1 && strings.EqualFold(text, "text") && strings.EqualFold(intent, "intent") {
			continue
		}
		samples = append(samples, Sample{Text: text, Intent: intent})
	}
	return samples, nil
}

// SaveSamplesCSV writes labeled samples into CSV with header text,intent.
func SaveSamplesCSV(path string, samples []Sample) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"text", "intent"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, s := range samples {
		text := strings.TrimSpace(s.Text)
		intent := strings.TrimSpace(s.Intent)
		if text == "" || intent == "" {
			continue
		}
		if err := w.Write([]string{text, intent}); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}
