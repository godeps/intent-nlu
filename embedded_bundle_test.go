package intentnlu

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedBundleManifest(t *testing.T) {
	manifest, err := EmbeddedBundleManifest()
	if err != nil {
		t.Fatalf("EmbeddedBundleManifest failed: %v", err)
	}
	if len(manifest.Models) == 0 {
		t.Fatalf("expected embedded models in manifest")
	}
	if strings.TrimSpace(manifest.DefaultLanguage) == "" {
		t.Fatalf("default language should not be empty")
	}
}

func TestExtractEmbeddedBundleAndLoadRouter(t *testing.T) {
	cacheDir := t.TempDir()
	bundleDir1, err := ExtractEmbeddedBundle(cacheDir)
	if err != nil {
		t.Fatalf("ExtractEmbeddedBundle failed: %v", err)
	}
	if !strings.HasPrefix(filepath.Clean(bundleDir1), filepath.Clean(cacheDir)) {
		t.Fatalf("expected bundle dir under cache dir, got %s", bundleDir1)
	}

	bundleDir2, err := ExtractEmbeddedBundle(cacheDir)
	if err != nil {
		t.Fatalf("ExtractEmbeddedBundle second call failed: %v", err)
	}
	if filepath.Clean(bundleDir1) != filepath.Clean(bundleDir2) {
		t.Fatalf("expected idempotent extract path, got %s vs %s", bundleDir1, bundleDir2)
	}

	router, err := NewRouterFromEmbeddedIn(cacheDir)
	if err != nil {
		t.Fatalf("NewRouterFromEmbeddedIn failed: %v", err)
	}
	langs := router.Languages()
	if len(langs) == 0 {
		t.Fatalf("expected router languages from embedded models")
	}

	pred, err := router.Predict(context.Background(), "hello", PredictOptions{
		TopK:         2,
		LanguageHint: "en",
	})
	if err != nil {
		t.Fatalf("embedded router predict failed: %v", err)
	}
	if strings.TrimSpace(pred.Language) == "" {
		t.Fatalf("expected non-empty prediction language")
	}
}

func TestNewRouterFromBundle_EmptyUsesEmbedded(t *testing.T) {
	router, err := NewRouterFromBundle("")
	if err != nil {
		t.Fatalf("NewRouterFromBundle empty failed: %v", err)
	}
	langs := router.Languages()
	if len(langs) == 0 {
		t.Fatalf("expected languages from embedded bundle")
	}
}
