package service

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeFile membuat file isi kecil lalu memperpanjangnya (sparse) ke ukuran n
// agar pengujian besar tidak memakan memori nyata.
func makeFile(t *testing.T, path string, n int64) {
	t.Helper()
	writeFile(t, path, 1)
	if err := os.Truncate(path, n); err != nil {
		t.Fatal(err)
	}
}

func ageFile(t *testing.T, path string, days int) {
	t.Helper()
	past := time.Now().AddDate(0, 0, -days)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
}

func TestFindBigFilesFilters(t *testing.T) {
	home := t.TempDir()
	dl := filepath.Join(home, "Downloads")
	desk := filepath.Join(home, "Desktop")

	mk := func(rel string, n int64, days int) string {
		p := filepath.Join(dl, rel)
		if rel[:3] == "dsk" {
			p = filepath.Join(desk, rel[4:])
		}
		makeFile(t, p, n)
		ageFile(t, p, days)
		return p
	}

	// (root, nama, ukuran, umur) → masuk/hit?
	oldZip := mk("sub/old-setup.zip", 300<<20, 60) // hit
	bigOld := mk("dsk:old-installer.rar", 120<<20, 45)
	_ = mk("recent-setup.zip", 110<<20, 2) // miss: masih baru
	_ = mk("small-junk.zip", 50<<20, 60)   // miss: < 100 MB default
	_ = mk("notes.txt", 200<<20, 60)       // miss: ekstensi tidak dikenal
	_ = mk("docs.dll", 200<<20, 60)        // miss: ekstensi tidak dikenal
	_ = mk("sub2/full.iso", 60<<20, 60)    // miss: < 100 MB
	if err := os.MkdirAll(filepath.Join(dl, "folder.zip"), 0755); err != nil {
		t.Fatal(err)
	}

	s := New(false)
	s.HomeDir = home
	files := s.FindBigFiles(home)

	if len(files) != 2 {
		t.Fatalf("FindBigFiles = %d file, ingin 2: %+v", len(files), files)
	}
	if files[0].Path != oldZip || files[1].Path != bigOld {
		t.Errorf("urutan/isi salah: %+v", files)
	}
	if files[0].Size != 300<<20 || files[1].Size != 120<<20 {
		t.Errorf("ukuran salah: %+v", files)
	}
	if files[0].Kind != "zip" || files[1].Kind != "rar" {
		t.Errorf("kind salah: %+v", files)
	}
}

func TestFindBigFilesHonorsKeepPatterns(t *testing.T) {
	home := t.TempDir()
	dl := filepath.Join(home, "Downloads")
	keep := filepath.Join(dl, "keep-me.zip")
	makeFile(t, keep, 200<<20)
	ageFile(t, keep, 60)
	drop := filepath.Join(dl, "drop-me.msi")
	makeFile(t, drop, 200<<20)
	ageFile(t, drop, 60)

	s := New(false)
	s.HomeDir = home
	s.KeepPatterns = []string{"keep-*"}
	files := s.FindBigFiles(home)
	if len(files) != 1 || files[0].Path != drop {
		t.Errorf("keep:* harus melindungi keep-me.zip, got %+v", files)
	}
}

func TestFindBigFilesMissingRootsOK(t *testing.T) {
	home := t.TempDir() // tanpa Downloads/Desktop sama sekali
	s := New(false)
	s.HomeDir = home
	if files := s.FindBigFiles(home); len(files) != 0 {
		t.Errorf("home kosong harus menghasilkan 0 file, got %+v", files)
	}
}

func TestDeleteBigFilesAccounting(t *testing.T) {
	home := t.TempDir()
	dl := filepath.Join(home, "Downloads")
	p1 := filepath.Join(dl, "a.zip")
	makeFile(t, p1, 100)
	p2 := filepath.Join(dl, "b.msi")
	makeFile(t, p2, 50)
	for _, p := range []string{p1, p2} {
		ageFile(t, p, 60)
	}

	s := New(false)
	s.HomeDir = home
	s.MinSize = 1
	files := s.FindBigFiles(home)
	if len(files) != 2 {
		t.Fatalf("harus ada 2 kandidat, got %d", len(files))
	}

	freed, deleted, failed := s.DeleteBigFiles(files)
	if freed != 150 || deleted != 2 || failed != 0 {
		t.Errorf("DeleteBigFiles = freed %d deleted %d failed %d, ingin 150/2/0",
			freed, deleted, failed)
	}
	if got := s.TotalFreed.Load(); got != 150 {
		t.Errorf("TotalFreed = %d, ingin 150", got)
	}
	for _, p := range []string{p1, p2} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("file %s harus sudah terhapus", p)
		}
	}
}

func TestDeleteBigFilesDryRunDoesNothing(t *testing.T) {
	home := t.TempDir()
	dl := filepath.Join(home, "Downloads")
	p := filepath.Join(dl, "a.zip")
	makeFile(t, p, 100)
	ageFile(t, p, 60)

	s := New(true)
	s.HomeDir = home
	s.MinSize = 1
	var buf bytes.Buffer
	s.Out = &buf
	files := s.FindBigFiles(home)
	if len(files) != 1 {
		t.Fatalf("harus ada 1 kandidat, got %d", len(files))
	}

	freed, deleted, failed := s.DeleteBigFiles(files)
	if freed != 0 || deleted != 0 || failed != 0 {
		t.Errorf("dry-run tidak boleh menghapus: freed=%d deleted=%d failed=%d", freed, deleted, failed)
	}
	if _, err := os.Stat(p); err != nil {
		t.Error("dry-run tidak boleh menghapus file")
	}
	if !bytes.Contains(buf.Bytes(), []byte("hapus file")) {
		t.Errorf("dry-run harus mencatat rencana, got:\\n%s", buf.String())
	}
}
