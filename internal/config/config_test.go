package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExtraPaths(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "paths.txt")

	content := "# Ini komentar\n\nold:/test/cache\n# Komentar lain\n/normal/path\n"
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	paths := LoadExtraPaths(configFile)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != "old:/test/cache" {
		t.Errorf("expected paths[0] == 'old:/test/cache', got %q", paths[0])
	}
	if paths[1] != "/normal/path" {
		t.Errorf("expected paths[1] == '/normal/path', got %q", paths[1])
	}
}

func TestTier1IncludesToolCaches(t *testing.T) {
	home := `C:\Users\Test`
	paths := DefaultTier1Paths(home)
	want := []string{
		filepath.Join(home, "AppData/Local/gopls"),
		filepath.Join(home, "AppData/Local/goimports"),
		filepath.Join(home, ".cache/puppeteer"),
		filepath.Join(home, ".cache/chrome-devtools-mcp"),
		filepath.Join(home, "AppData/Local/ms-playwright"),
	}
	for _, w := range want {
		found := false
		for _, p := range paths {
			if p == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultTier1Paths harus memuat %q", w)
		}
	}
}

func TestLoadKeepPatterns(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "cfg")
	content := "# komentar\nkeep:*.tmp\nold:/x\nkeep: backup\\*\n"
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	pats := LoadKeepPatterns(cfg)
	if len(pats) != 2 || pats[0] != "*.tmp" || pats[1] != "backup\\*" {
		t.Errorf("LoadKeepPatterns = %v, ingin [*.tmp backup\\*]", pats)
	}
}
