package repository

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrNotDir = errors.New("not a directory")

func IsNotExist(err error) bool { return os.IsNotExist(err) }

func StatIsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, ErrNotDir
	}
	return true, nil
}

func DirSize(path string) int64 {
	var size int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			size += info.Size()
		}
		return nil
	})
	return size
}

func RemoveTree(path string) error { return os.RemoveAll(path) }

func RemoveFile(path string) error { return os.Remove(path) }

func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Children(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func Glob(pattern string) []string {
	m, _ := filepath.Glob(pattern)
	return m
}

func Walk(dir string, fn func(path string, d fs.DirEntry, err error) error) error {
	return filepath.WalkDir(dir, fn)
}
