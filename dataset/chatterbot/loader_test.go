package chatterbot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSamples(t *testing.T) {
	root := t.TempDir()
	greetings := `categories:
  - greetings
conversations:
  - - 你好
    - 你好呀
  - - 早上好
    - 早
`
	calendar := `categories:
  - calendar
conversations:
  - - 明天星期几
    - 我帮你查下
  - 今天是个好日子
`
	if err := os.WriteFile(filepath.Join(root, "greetings.yml"), []byte(greetings), 0o644); err != nil {
		t.Fatalf("write greetings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "calendar.yml"), []byte(calendar), 0o644); err != nil {
		t.Fatalf("write calendar: %v", err)
	}

	samples, err := LoadSamples(root, LoaderOptions{
		FileIntent: map[string]string{
			"greetings.yml": "chitchat_greeting",
			"calendar.yml":  "calendar_info",
		},
		SkipUnmapped:   true,
		IncludeReplies: false,
	})
	if err != nil {
		t.Fatalf("LoadSamples failed: %v", err)
	}
	if len(samples) != 4 {
		t.Fatalf("expected 4 samples, got %d", len(samples))
	}
	if samples[0].Intent == "" || samples[0].Text == "" {
		t.Fatalf("sample should be non-empty: %+v", samples[0])
	}
}

func TestNormalizeConversation(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int
	}{
		{name: "string", in: "hello", want: 1},
		{name: "slice-string", in: []string{"a", " ", "b"}, want: 2},
		{name: "slice-any", in: []any{"a", 1, nil, "b"}, want: 3},
		{name: "invalid", in: nil, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeConversation(tc.in)
			if len(got) != tc.want {
				t.Fatalf("expected %d items, got %d (%v)", tc.want, len(got), got)
			}
		})
	}
}

func TestLoadFileIntentMap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.yaml")
	content := "greetings.yml: chitchat_greeting\ncalendar.yml: calendar_info\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write map file: %v", err)
	}
	mapping, err := LoadFileIntentMap(path)
	if err != nil {
		t.Fatalf("LoadFileIntentMap failed: %v", err)
	}
	if mapping["greetings.yml"] != "chitchat_greeting" {
		t.Fatalf("unexpected map result: %#v", mapping)
	}
}
