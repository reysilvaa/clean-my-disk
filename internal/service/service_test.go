package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"clean-my-disk/internal/model"
)

func writeFile(t *testing.T, path string, n int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCleanOldRootProtectionAndAccounting(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "updater_test")
	if err := os.MkdirAll(filepath.Join(targetDir, "sub_empty"), 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	oldFile := filepath.Join(targetDir, "old_file.bin")
	writeFile(t, oldFile, 100)
	pastTime := time.Now().AddDate(0, 0, -30)
	_ = os.Chtimes(oldFile, pastTime, pastTime)

	newFile := filepath.Join(targetDir, "new_file.bin")
	writeFile(t, newFile, 200)

	svc := New(false)
	threshold := time.Now().AddDate(0, 0, -model.OldFileThresholdDays)

	var wg sync.WaitGroup
	wg.Add(1)
	svc.CleanOld(targetDir, threshold, &wg)
	wg.Wait()

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Error("Root target directory was mistakenly deleted")
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Old file was NOT deleted")
	}
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("New file was mistakenly deleted")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "sub_empty")); !os.IsNotExist(err) {
		t.Error("Empty subdirectory was not cleaned up")
	}
	if freed := svc.TotalFreed.Load(); freed != 100 {
		t.Errorf("TotalFreed = %d, expected exactly 100 bytes (only the old file)", freed)
	}
}

func TestWipeContents(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "temp_cache")
	writeFile(t, filepath.Join(targetDir, "file1.txt"), 50)
	writeFile(t, filepath.Join(targetDir, "sub", "file2.txt"), 50)

	svc := New(false)
	var wg sync.WaitGroup
	wg.Add(1)
	svc.WipeContents(targetDir, &wg)
	wg.Wait()

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Fatal("Root directory was deleted")
	}
	entries, _ := os.ReadDir(targetDir)
	if len(entries) != 0 {
		t.Errorf("Target directory has %d remaining entries, expected 0", len(entries))
	}
	if freed := svc.TotalFreed.Load(); freed != 100 {
		t.Errorf("TotalFreed = %d, expected 100 bytes", freed)
	}
}

func TestWipeContentsMinSize(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a"), 100)

	svc := New(false)
	svc.MinSize = 1 << 20
	var wg sync.WaitGroup
	wg.Add(1)
	svc.WipeContents(dir, &wg)
	wg.Wait()

	if got := svc.TotalFreed.Load(); got != 0 {
		t.Errorf("TotalFreed = %d, ingin 0 (target kecil harus dilewati)", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "a")); err != nil {
		t.Error("file tidak boleh dihapus pada target yang dilewati")
	}
}

func TestKeepPatterns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "keep.tmp"), 100)
	writeFile(t, filepath.Join(dir, "drop.txt"), 100)

	svc := New(false)
	svc.KeepPatterns = []string{"*.tmp"}
	var wg sync.WaitGroup
	wg.Add(1)
	svc.WipeContents(dir, &wg)
	wg.Wait()

	if _, err := os.Stat(filepath.Join(dir, "keep.tmp")); err != nil {
		t.Error("keep.tmp harus dipertahankan (keep:*.tmp)")
	}
	if _, err := os.Stat(filepath.Join(dir, "drop.txt")); !os.IsNotExist(err) {
		t.Error("drop.txt harus dihapus")
	}
	if got := svc.TotalFreed.Load(); got != 100 {
		t.Errorf("TotalFreed = %d, ingin 100 (hanya drop.txt)", got)
	}
}

func TestBrowserCacheFindsProfiles(t *testing.T) {
	ud := t.TempDir()
	writeFile(t, filepath.Join(ud, "Default", "Cache", "a"), 50)
	writeFile(t, filepath.Join(ud, "Profile 1", "Code Cache", "b"), 60)
	writeFile(t, filepath.Join(ud, "Profile 2", "Cache", "c"), 70)
	writeFile(t, filepath.Join(ud, "Guest", "Cache", "d"), 80)

	svc := New(false)
	var wg sync.WaitGroup
	wg.Add(1)
	svc.BrowserCache(ud, "Chrome", &wg)
	wg.Wait()

	if got := svc.TotalFreed.Load(); got != 50+60+70 {
		t.Errorf("TotalFreed = %d, ingin %d (Guest harus diabaikan)", got, 50+60+70)
	}
	if _, err := os.Stat(filepath.Join(ud, "Default", "Cache")); !os.IsNotExist(err) {
		t.Error("Default/Cache harus sudah dihapus")
	}
	if _, err := os.Stat(filepath.Join(ud, "Guest", "Cache", "d")); err != nil {
		t.Error("Guest (bukan profil standar) tidak boleh disentuh")
	}
}

func TestRunTier2CargoKeepsIndex(t *testing.T) {
	home := t.TempDir()
	reg := filepath.Join(home, ".cargo", "registry")
	writeFile(t, filepath.Join(reg, "cache", "x", "f"), 100)
	writeFile(t, filepath.Join(reg, "src", "y", "g"), 100)
	writeFile(t, filepath.Join(reg, "index", "keep.me"), 30)

	svc := New(false)
	svc.HomeDir = home
	svc.RunTier2()

	if _, err := os.Stat(filepath.Join(reg, "index", "keep.me")); err != nil {
		t.Error("cargo index ikut terhapus — harus dipertahankan")
	}
	for _, sub := range []string{"cache", "src"} {
		entries, err := os.ReadDir(filepath.Join(reg, sub))
		if err != nil {
			t.Fatalf("folder %s harus tetap ada (hanya isinya yang dihapus): %v", sub, err)
		}
		if len(entries) != 0 {
			t.Errorf("folder %s harus kosong, masih ada %d entri", sub, len(entries))
		}
	}
	if got := svc.TotalFreed.Load(); got != 200 {
		t.Errorf("TotalFreed = %d, ingin 200", got)
	}
}

func TestJunctionWipeLeavesExternalTarget(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	external := filepath.Join(root, "external")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(external, 0755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(external, "precious.txt")
	secretData := []byte("critical data")
	if err := os.WriteFile(secret, secretData, 0644); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(cacheDir, "junction_point")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, external).CombinedOutput(); err != nil {
		t.Skipf("mklink gagal (%v: %s)", err, out)
	}

	svc := New(false)
	var wg sync.WaitGroup
	wg.Add(1)
	svc.WipeContents(cacheDir, &wg)
	wg.Wait()

	if _, err := os.Lstat(junction); !os.IsNotExist(err) {
		t.Error("junction link harus dihapus dari area wipe")
	}
	data, err := os.ReadFile(secret)
	if err != nil {
		t.Fatalf("SECURITY VIOLATION: target eksternal junction ikut terhapus: %v", err)
	}
	if string(data) != string(secretData) {
		t.Fatal("SECURITY VIOLATION: file eksternal rusak")
	}
}

func TestScanAll(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, "AppData", "Local", "Temp", "junk"), 10)
	extra := t.TempDir()
	writeFile(t, filepath.Join(extra, "f"), 5)

	items := ScanAll(home, []string{extra})
	if len(items) != 2 {
		t.Fatalf("ScanAll = %d item, ingin 2 (Temp + extra): %+v", len(items), items)
	}
	var tempSz, extraSz int64
	for _, it := range items {
		switch it.Kind {
		case "cache":
			tempSz += it.Size
		case "extra":
			extraSz += it.Size
		}
	}
	if tempSz != 10 || extraSz != 5 {
		t.Errorf("ukuran scan salah: temp=%d extra=%d, ingin 10 dan 5", tempSz, extraSz)
	}
}
