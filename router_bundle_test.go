package chatnlu

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRouterFromBundle(t *testing.T) {
	zhDir := trainModelForTest(t, "zh", []Sample{
		{Text: "明天星期几", Intent: "calendar_info"},
		{Text: "北京会下雨吗", Intent: "weather_info"},
	})
	enDir := trainModelForTest(t, "en", []Sample{
		{Text: "what day is tomorrow", Intent: "calendar_info"},
		{Text: "will it rain in beijing", Intent: "weather_info"},
	})

	bundleDir := t.TempDir()
	zhBundleModel := filepath.Join(bundleDir, "models", "zh")
	enBundleModel := filepath.Join(bundleDir, "models", "en")
	copyModelDir(t, zhDir, zhBundleModel)
	copyModelDir(t, enDir, enBundleModel)

	err := SaveBundleManifest(bundleDir, BundleManifest{
		Version:         "bundle-test-v1",
		DefaultLanguage: "en",
		Models: map[string]string{
			"zh": "models/zh",
			"en": "models/en",
		},
	})
	if err != nil {
		t.Fatalf("SaveBundleManifest failed: %v", err)
	}

	router, err := NewRouterFromBundle(bundleDir)
	if err != nil {
		t.Fatalf("NewRouterFromBundle failed: %v", err)
	}

	pred, err := router.Predict(context.Background(), "明天是星期几", PredictOptions{TopK: 2, LanguageHint: "auto"})
	if err != nil {
		t.Fatalf("router predict failed: %v", err)
	}
	if pred.Language != "zh" {
		t.Fatalf("expected zh prediction, got %s", pred.Language)
	}
}

func copyModelDir(t *testing.T, srcDir string, dstDir string) {
	t.Helper()
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	for _, name := range []string{ModelBinaryFile, ModelMetaFile} {
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(dstDir, name)
		bytes, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s failed: %v", src, err)
		}
		if err := os.WriteFile(dst, bytes, 0o644); err != nil {
			t.Fatalf("write %s failed: %v", dst, err)
		}
	}
}
