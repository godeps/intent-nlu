package intentnlu

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

const embeddedBundleRoot = "models/multilingual"

//go:embed models/multilingual
var embeddedBundleFS embed.FS

var (
	embeddedExtractMu    sync.Mutex
	embeddedExtractCache = map[string]string{}
)

// NewRouterFromEmbedded loads router from embedded multilingual bundle assets.
func NewRouterFromEmbedded() (*Router, error) {
	return NewRouterFromEmbeddedIn("")
}

// NewRouterFromEmbeddedIn loads router from embedded assets and extracts files under cacheDir.
// If cacheDir is empty, user cache directory (or system temp) is used.
func NewRouterFromEmbeddedIn(cacheDir string) (*Router, error) {
	bundleDir, err := ExtractEmbeddedBundle(cacheDir)
	if err != nil {
		return nil, err
	}
	return NewRouterFromBundle(bundleDir)
}

// EmbeddedBundleManifest reads bundle manifest from embedded assets.
func EmbeddedBundleManifest() (BundleManifest, error) {
	manifestPath := path.Join(embeddedBundleRoot, BundleManifestFileName)
	bytes, err := fs.ReadFile(embeddedBundleFS, manifestPath)
	if err != nil {
		return BundleManifest{}, fmt.Errorf("read embedded bundle manifest: %w", err)
	}
	var manifest BundleManifest
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		return BundleManifest{}, fmt.Errorf("parse embedded bundle manifest: %w", err)
	}
	if len(manifest.Models) == 0 {
		return BundleManifest{}, fmt.Errorf("embedded bundle manifest has empty models")
	}
	return manifest, nil
}

// ExtractEmbeddedBundle extracts embedded multilingual bundle to local filesystem and returns bundle directory.
// Extraction is idempotent for the same cacheDir and embedded manifest hash.
func ExtractEmbeddedBundle(cacheDir string) (string, error) {
	manifestPath := path.Join(embeddedBundleRoot, BundleManifestFileName)
	manifestBytes, err := fs.ReadFile(embeddedBundleFS, manifestPath)
	if err != nil {
		return "", fmt.Errorf("read embedded manifest: %w", err)
	}
	hash := sha256.Sum256(manifestBytes)
	hashText := hex.EncodeToString(hash[:8])

	baseDir := strings.TrimSpace(cacheDir)
	if baseDir == "" {
		baseDir = defaultEmbeddedCacheDir()
	}
	targetDir := filepath.Join(baseDir, "embedded-bundles", hashText)
	key := filepath.Clean(targetDir)

	embeddedExtractMu.Lock()
	defer embeddedExtractMu.Unlock()
	if cached, ok := embeddedExtractCache[key]; ok && cached != "" {
		if _, statErr := os.Stat(filepath.Join(cached, BundleManifestFileName)); statErr == nil {
			return cached, nil
		}
	}
	if _, statErr := os.Stat(filepath.Join(targetDir, BundleManifestFileName)); statErr == nil {
		embeddedExtractCache[key] = targetDir
		return targetDir, nil
	}
	if err := extractEmbeddedBundleToDir(targetDir); err != nil {
		return "", err
	}
	embeddedExtractCache[key] = targetDir
	return targetDir, nil
}

func extractEmbeddedBundleToDir(targetDir string) error {
	subFS, err := fs.Sub(embeddedBundleFS, embeddedBundleRoot)
	if err != nil {
		return fmt.Errorf("create embedded sub fs: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create embedded target dir: %w", err)
	}
	err = fs.WalkDir(subFS, ".", func(name string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		dst := filepath.Join(targetDir, filepath.FromSlash(name))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, readErr := fs.ReadFile(subFS, name)
		if readErr != nil {
			return readErr
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("extract embedded bundle: %w", err)
	}
	return nil
}

func defaultEmbeddedCacheDir() string {
	userCacheDir, err := os.UserCacheDir()
	if err == nil && strings.TrimSpace(userCacheDir) != "" {
		return filepath.Join(userCacheDir, "intent-nlu")
	}
	return filepath.Join(os.TempDir(), "intent-nlu")
}
