package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func createJunction(t *testing.T, link, target string) (skip bool) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Logf("mklink gagal: %v (%s) — lewati", err, out)
		return true
	}
	return false
}

func TestRemoveTreeJunctionSafety(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(target, "secret.txt")
	if err := os.WriteFile(secret, []byte("data target jangan hilang"), 0644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(base, "link")
	if createJunction(t, link, target) {
		t.Skip("junction tidak tersedia")
	}

	if err := RemoveTree(link); err != nil {
		t.Fatalf("RemoveTree junction gagal: %v", err)
	}
	if _, err := os.Stat(link); !os.IsNotExist(err) {
		t.Errorf("junction masih ada setelah RemoveTree")
	}
	if _, err := os.Stat(secret); err != nil {
		t.Errorf("isi target ikut terhapus: %v", err)
	}
}

func TestDirSizeDoesNotFollowJunction(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "base")
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "secret.bin"), make([]byte, 500), 0644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(base, "link")
	if createJunction(t, link, target) {
		t.Skip("junction tidak tersedia")
	}

	if sz := DirSize(base); sz != 0 {
		t.Errorf("DirSize(base) = %d, ingin 0 (junction tidak boleh ditembus)", sz)
	}
}

func TestGetDriveSpace(t *testing.T) {
	free, total, pct := GetDriveSpace()
	if total <= 0 {
		t.Skipf("tidak ada drive fixed terukur (total=%d)", total)
	}
	if free < 0 || free > total {
		t.Errorf("free=%d di luar rentang [0,total=%d]", free, total)
	}
	if pct < 0 || pct > 100 {
		t.Errorf("percentUsed=%.1f di luar [0,100]", pct)
	}
}
