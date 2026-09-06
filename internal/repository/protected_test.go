package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func setupProtectedEnv(t *testing.T) (root, home string) {
	t.Helper()
	root = t.TempDir()
	sys := filepath.Join(root, "Windows")
	pf := filepath.Join(root, "Program Files")
	pfx := filepath.Join(root, "Program Files (x86)")
	pd := filepath.Join(root, "ProgramData")
	t.Setenv("SystemRoot", sys)
	t.Setenv("ProgramFiles", pf)
	t.Setenv("ProgramFiles(x86)", pfx)
	t.Setenv("ProgramData", pd)
	return root, filepath.Join(root, "Users", "Me")
}

func TestIsProtectedPathDenies(t *testing.T) {
	root, home := setupProtectedEnv(t)
	cases := []string{
		root + `\`,                     // akar drive
		filepath.Join(root, "Windows"), // SystemRoot
		filepath.Join(root, "Windows", "System32"),
		filepath.Join(root, "Program Files"),
		filepath.Join(root, "Program Files", "App"),
		filepath.Join(root, "Program Files (x86)", "App"),
		filepath.Join(root, "ProgramData"),
		filepath.Join(root, "ProgramData", "NVIDIA"),
		home,                         // profil user
		filepath.Join(root, "Users"), // induk profil
	}
	for _, p := range cases {
		if !IsProtectedPath(p, home) {
			t.Errorf("harus DITOLAK: %s", p)
		}
	}
}

func TestIsProtectedPathAllows(t *testing.T) {
	root, home := setupProtectedEnv(t)
	cases := []string{
		filepath.Join(home, "AppData", "Local", "Temp"),
		filepath.Join(home, "AppData", "Local", "npm-cache"),
		filepath.Join(home, "Documents"),
		filepath.Join(root, "Users", "Lain", "AppData", "Local", "Temp"),
		filepath.Join(root, "Data", "Downloads"),
		filepath.Join(root, "Data", "Downloads", "Lama"),
	}
	for _, p := range cases {
		if IsProtectedPath(p, home) {
			t.Errorf("harus DIIZINKAN: %s", p)
		}
	}
}

func TestIsProtectedPathEnvFallback(t *testing.T) {
	root, home := setupProtectedEnv(t)
	os.Unsetenv("ProgramFiles")
	if !IsProtectedPath(filepath.Join(root, "Windows", "Temp"), home) {
		t.Error("SystemRoot tetap harus dilindungi walau ProgramFiles kosong")
	}
}
