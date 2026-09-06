package service

import (
	"path/filepath"
	"strings"

	"clean-my-disk/internal/config"
	"clean-my-disk/internal/model"
	"clean-my-disk/internal/repository"
)

func ScanAll(home string, extras []string) []model.ScanItem {
	var items []model.ScanItem
	seen := map[string]bool{}
	add := func(p, kind string) {
		if !repository.IsDir(p) || seen[p] {
			return
		}
		seen[p] = true
		items = append(items, model.ScanItem{Path: p, Size: repository.DirSize(p), Kind: kind})
	}
	for _, p := range config.DefaultTier1Paths(home) {
		add(p, "cache")
	}
	for name, dir := range config.BrowserUserDirs(home) {
		sz := int64(0)
		for _, t := range findCacheTargets(dir, name) {
			sz += repository.DirSize(t)
		}
		if sz > 0 {
			items = append(items, model.ScanItem{Path: name + " caches", Size: sz, Kind: "browser"})
		}
	}
	for _, p := range config.DefaultUpdaterPaths(home) {
		add(p, "updater")
	}
	for _, raw := range extras {
		p := strings.TrimSpace(raw)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		p = strings.TrimPrefix(p, "old:")
		if repository.IsProtectedPath(p, home) {
			continue
		}
		add(p, "extra")
	}
	return items
}

func TierEstimates(home string) []int64 {
	est := make([]int64, 6)
	for _, p := range config.DefaultTier1Paths(home) {
		est[0] += repository.DirSize(p)
	}
	for name, dir := range config.BrowserUserDirs(home) {
		for _, t := range findCacheTargets(dir, name) {
			est[1] += repository.DirSize(t)
		}
	}
	for _, p := range config.DefaultUpdaterPaths(home) {
		est[2] += repository.DirSize(p)
	}
	est[3] = repository.DirSize(config.GradleCacheDir(home)) +
		repository.DirSize(filepath.Join(config.CargoRegistryDir(home), "cache")) +
		repository.DirSize(filepath.Join(config.CargoRegistryDir(home), "src"))
	for _, p := range config.DefaultTier3Paths() {
		est[4] += repository.DirSize(p)
	}

	return est
}

func Snapshot() model.SystemInfo {
	drives := repository.FixedDrives()
	free, total, pct := model.AggregateDrives(drives)
	return model.SystemInfo{
		Drives:     drives,
		DriveFree:  free,
		DriveTotal: total,
		UsedPct:    pct,
		IsAdmin:    repository.IsAdmin(),
	}
}
