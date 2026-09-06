package service

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"clean-my-disk/internal/config"
	"clean-my-disk/internal/i18n"
	"clean-my-disk/internal/model"
	"clean-my-disk/internal/repository"
)

type Service struct {
	DryRun        bool
	HomeDir       string
	Lang          string
	ThresholdDays int
	MinSize       int64
	KeepPatterns  []string
	Trash         bool
	Out           io.Writer
	TotalFreed    atomic.Int64
	Failures      atomic.Int64
	logMu         sync.Mutex
	OnLog         func(target, status string)
}

func New(dryRun bool) *Service {
	return &Service{
		DryRun:  dryRun,
		HomeDir: config.Home(),
	}
}

func (s *Service) t(key string) string { return i18n.T(s.Lang, key) }
func (s *Service) tf(key string, a ...any) string {
	return fmt.Sprintf(i18n.T(s.Lang, key), a...)
}

func (s *Service) thresholdDays() int {
	days := s.ThresholdDays
	if days <= 0 {
		days = model.OldFileThresholdDays
	}
	return days
}

func (s *Service) threshold() time.Time {
	return time.Now().AddDate(0, 0, -s.thresholdDays())
}

func (s *Service) keepMatch(p string) bool {
	base := filepath.Base(p)
	for _, pat := range s.KeepPatterns {
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, p); ok {
			return true
		}
	}
	return false
}

func (s *Service) errStatus(err error) string {
	switch {
	case err == nil:
		return ""
	case repository.IsNotExist(err):
		return s.t("not_exists")
	case errors.Is(err, repository.ErrNotDir):
		return s.t("not_dir")
	default:
		return s.t("access_denied")
	}
}

func (s *Service) Log(target, status string) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	if s.OnLog != nil {
		s.OnLog(target, status)
		return
	}
	if s.Out != nil {
		fmt.Fprintf(s.Out, "%-62s %s\n", target, status)
	}
}

func (s *Service) header(title string) {
	if s.OnLog != nil || s.Out == nil {
		return
	}
	fmt.Fprintln(s.Out, "----------------------------------------------")
	fmt.Fprintln(s.Out, title)
}

func (s *Service) raw(line string) {
	if s.OnLog != nil || s.Out == nil {
		return
	}
	fmt.Fprintln(s.Out, line)
}

func (s *Service) WipeContents(dir string, wg *sync.WaitGroup) {
	defer wg.Done()
	entries, err := repository.Children(dir)
	if err != nil {
		s.Log(dir, s.errStatus(err))
		return
	}
	before := repository.DirSize(dir)

	if s.MinSize > 0 && before < s.MinSize {
		s.Log(dir, s.tf("skip_small", model.HumanSize(s.MinSize)))
		return
	}

	if s.DryRun {
		s.Log(dir, s.tf("wipe_plan", model.HumanSize(before)))
		return
	}

	var failed int
	for _, name := range entries {
		p := filepath.Join(dir, name)
		if s.keepMatch(p) {
			continue
		}
		if err := repository.RemoveTree(p); err != nil {
			failed++
		}
	}

	after := repository.DirSize(dir)
	freed := before - after
	if freed < 0 {
		freed = 0
	}
	s.TotalFreed.Add(freed)
	s.Failures.Add(int64(failed))

	status := s.tf("ok_size", model.HumanSize(freed))
	if failed > 0 {
		status = s.tf("ok_size_failed", model.HumanSize(freed), failed)
	}
	s.Log(dir, status)
}

func (s *Service) CleanOld(dir string, threshold time.Time, wg *sync.WaitGroup) {
	defer wg.Done()
	if _, err := repository.StatIsDir(dir); err != nil {
		s.Log(dir, s.errStatus(err))
		return
	}
	if s.MinSize > 0 && repository.DirSize(dir) < s.MinSize {
		s.Log(dir, s.tf("skip_small", model.HumanSize(s.MinSize)))
		return
	}

	var targetFiles []string
	var targetSizes []int64
	var emptyDirs []string

	walkErr := repository.Walk(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if s.keepMatch(p) {
			return nil
		}
		if !d.IsDir() {
			if info.ModTime().Before(threshold) {
				targetFiles = append(targetFiles, p)
				targetSizes = append(targetSizes, info.Size())
			}
		} else {
			emptyDirs = append(emptyDirs, p)
		}
		return nil
	})
	if walkErr != nil {
		s.Log(dir, s.errStatus(walkErr))
		return
	}

	var candidate int64
	for _, sz := range targetSizes {
		candidate += sz
	}

	if s.DryRun {
		s.Log(dir, s.tf("old_plan", model.HumanSize(candidate), len(targetFiles)))
		return
	}

	var freed int64
	var failed int
	for i, f := range targetFiles {
		if err := repository.RemoveFile(f); err == nil {
			freed += targetSizes[i]
		} else {
			failed++
		}
	}

	var emptyFailed int
	for i := len(emptyDirs) - 1; i >= 0; i-- {
		if err := repository.RemoveFile(emptyDirs[i]); err != nil {
			emptyFailed++
		}
	}

	s.TotalFreed.Add(freed)
	s.Failures.Add(int64(failed + emptyFailed))
	status := s.tf("ok_files", model.HumanSize(freed), len(targetFiles)-failed)
	if failed > 0 {
		status = s.tf("ok_files_failed", model.HumanSize(freed), failed)
	}
	if emptyFailed > 0 {
		status += s.tf("empty_dir_fail", emptyFailed)
	}
	s.Log(dir, status)
}

func findCacheTargets(userData, name string) []string {
	targets := config.BrowserCacheTargets
	profileGlob := filepath.Join(userData, "*")
	standard := func(base string) bool {
		return base == "Default" || strings.HasPrefix(base, "Profile ")
	}
	if name == "Firefox" {
		targets = config.FirefoxCacheTargets
		profileGlob = filepath.Join(userData, "Profiles", "*")
		standard = func(string) bool { return true }
	}

	var found []string
	for _, t := range targets {
		p := filepath.Join(userData, t)
		if repository.Exists(p) {
			found = append(found, p)
		}
	}
	for _, prof := range repository.Glob(profileGlob) {
		if !standard(filepath.Base(prof)) {
			continue
		}
		for _, t := range targets {
			p := filepath.Join(prof, t)
			if repository.Exists(p) {
				found = append(found, p)
			}
		}
	}
	return found
}

func (s *Service) BrowserCache(userData, name string, wg *sync.WaitGroup) {
	defer wg.Done()
	if _, err := repository.StatIsDir(userData); err != nil {
		s.Log(name, s.errStatus(err))
		return
	}

	found := findCacheTargets(userData, name)
	if len(found) == 0 {
		s.Log(name, s.t("cache_empty"))
		return
	}

	if s.DryRun {
		var totalEst int64
		for _, f := range found {
			totalEst += repository.DirSize(f)
		}
		s.Log(name, s.tf("browser_plan", model.HumanSize(totalEst)))
		return
	}

	var freed int64
	var failed int
	for _, f := range found {
		before := repository.DirSize(f)
		if err := repository.RemoveTree(f); err == nil {
			freed += before
		} else {
			failed++
			if after := repository.DirSize(f); before > after {
				freed += before - after
			}
		}
	}
	s.TotalFreed.Add(freed)
	s.Failures.Add(int64(failed))
	status := s.tf("ok_size", model.HumanSize(freed))
	if failed > 0 {
		status = s.tf("ok_locked", model.HumanSize(freed), failed)
	}
	s.Log(name, status)
}

func (s *Service) RemoveFile(path string, wg *sync.WaitGroup) {
	defer wg.Done()
	sz, err := repository.FileSize(path)
	if err != nil {
		s.Log(path, s.errStatus(err))
		return
	}
	if s.DryRun {
		s.Log(path, s.tf("file_plan", model.HumanSize(sz)))
		return
	}
	if err := repository.RemoveFile(path); err != nil {
		s.Failures.Add(1)
		s.Log(path, s.t("remove_fail"))
		return
	}
	s.TotalFreed.Add(sz)
	s.Log(path, s.tf("ok_size", model.HumanSize(sz)))
}

func (s *Service) TrashPath(path string, wg *sync.WaitGroup) {
	defer wg.Done()
	sz, err := repository.FileSize(path)
	if err != nil {
		s.Log(path, s.errStatus(err))
		return
	}
	if s.DryRun {
		s.Log(path, s.tf("trash_plan", model.HumanSize(sz)))
		return
	}
	if err := repository.MoveToTrash(path); err != nil {
		s.Failures.Add(1)
		s.Log(path, s.t("remove_fail"))
		return
	}
	s.TotalFreed.Add(sz)
	s.Log(path, s.tf("ok_size", model.HumanSize(sz)))
}

func (s *Service) TrashDir(dir string, wg *sync.WaitGroup) {
	defer wg.Done()
	entries, err := repository.Children(dir)
	if err != nil {
		s.Log(dir, s.errStatus(err))
		return
	}
	before := repository.DirSize(dir)
	if s.DryRun {
		s.Log(dir, s.tf("trash_plan", model.HumanSize(before)))
		return
	}
	var failed int
	for _, name := range entries {
		p := filepath.Join(dir, name)
		if s.keepMatch(p) {
			continue
		}
		if err := repository.MoveToTrash(p); err != nil {
			failed++
		}
	}
	after := repository.DirSize(dir)
	freed := before - after
	if freed < 0 {
		freed = 0
	}
	s.TotalFreed.Add(freed)
	s.Failures.Add(int64(failed))
	status := s.tf("ok_size", model.HumanSize(freed))
	if failed > 0 {
		status = s.tf("ok_size_failed", model.HumanSize(freed), failed)
	}
	s.Log(dir, status)
}

func (s *Service) RecycleBin() {
	if s.DryRun {
		s.Log(s.t("recycle_title"), s.t("recycle_plan"))
		return
	}
	beforeFree, _, _ := repository.GetDriveSpace()
	if err := repository.EmptyRecycleBin(); err != nil {
		s.Failures.Add(1)
		s.Log(s.t("recycle_title"), s.t("recycle_fail"))
		return
	}
	afterFree, _, _ := repository.GetDriveSpace()

	freed := afterFree - beforeFree
	if freed < 0 {
		freed = 0
	}
	s.TotalFreed.Add(freed)
	s.Log(s.t("recycle_title"), s.tf("ok_size", model.HumanSize(freed)))
}
