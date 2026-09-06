package service

import (
	"bytes"
	"os"
	"path/filepath"
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
