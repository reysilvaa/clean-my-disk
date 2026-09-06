package service

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTier1WritesThroughOut(t *testing.T) {
	home := t.TempDir()
	s := New(true)
	s.HomeDir = home
	s.Lang = "id"
	var buf bytes.Buffer
	s.Out = &buf
	s.RunTier1Regenerable()

	if !bytes.Contains(buf.Bytes(), []byte("TIER 1: cache regenerable")) {
		t.Errorf("header tier harus lewat Out, got:\n%s", buf.String())
	}
}

func TestLogSilentWithoutOutAndOnLog(t *testing.T) {
	s := New(true)
	before := os.Stdout
	defer func() { os.Stdout = before }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	s.Log("target-x", "OK (~1 B)")
	_ = w.Close()
	os.Stdout = before

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	if n > 0 {
		t.Errorf("tanpa Out/OnLog tidak boleh mencetak ke stdout, got: %q", buf[:n])
	}
}

func TestTier1WipesGoToolCaches(t *testing.T) {
	home := t.TempDir()
	for _, p := range []string{"AppData/Local/gopls", "AppData/Local/goimports"} {
		dir := filepath.Join(home, filepath.FromSlash(p))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cache.bin"), []byte("junk"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := New(true)
	s.HomeDir = home
	s.Lang = "id"
	var buf bytes.Buffer
	s.Out = &buf
	s.RunTier1Regenerable()

	out := buf.String()
	for _, p := range []string{"gopls", "goimports"} {
		if !strings.Contains(out, p) {
			t.Errorf("dry-run harus menyebut %s, got:\n%s", p, out)
		}
	}
}

func TestTier1WipesBrowserDownloadCaches(t *testing.T) {
	home := t.TempDir()
	for _, p := range []string{"AppData/Local/ms-playwright", ".cache/puppeteer", ".cache/chrome-devtools-mcp"} {
		dir := filepath.Join(home, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Join(dir, "chrome-headless-shell"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "chrome-headless-shell", "chrome.exe"), []byte("browser-binary"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := New(true)
	s.HomeDir = home
	s.Lang = "en"
	var buf bytes.Buffer
	s.Out = &buf
	s.RunTier1Regenerable()

	out := buf.String()
	for _, p := range []string{"puppeteer", "chrome-devtools-mcp", "ms-playwright"} {
		if !strings.Contains(out, p) {
			t.Errorf("dry-run harus menyebut %s, got:\n%s", p, out)
		}
	}
}

func TestHeaderUsesOutNotStdout(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, "tidak-ada"), 0755)
	s := New(true)
	s.HomeDir = home
	s.Lang = "en"
	var buf bytes.Buffer
	s.Out = &buf
	s.RunTier1Browsers()

	if bytes.Contains(buf.Bytes(), []byte("=== TIER 1: browser")) == false {
		t.Errorf("header tier1b harus lewat Out:\n%s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("Chrome")) {
		t.Errorf("baris target browser harus ada di Out:\n%s", buf.String())
	}
}
