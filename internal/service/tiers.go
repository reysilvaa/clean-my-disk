package service

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"clean-my-disk/internal/config"
	"clean-my-disk/internal/model"
	"clean-my-disk/internal/repository"
)

func (s *Service) RunTier1Regenerable() {
	s.raw(s.t("tier1a"))
	var wg sync.WaitGroup
	for _, p := range config.DefaultTier1Paths(s.HomeDir) {
		wg.Add(1)
		go s.WipeContents(p, &wg)
	}
	wg.Wait()
}

func (s *Service) RunTier1Browsers() {
	s.header(s.t("tier1b"))
	var wg sync.WaitGroup
	for name, dir := range config.BrowserUserDirs(s.HomeDir) {
		wg.Add(1)
		go s.BrowserCache(dir, name, &wg)
	}
	wg.Wait()
}

func (s *Service) RunTier1Updaters() {
	s.header(s.tf("tier1c", s.thresholdDays()))
	var wg sync.WaitGroup
	threshold := s.threshold()
	for _, u := range config.DefaultUpdaterPaths(s.HomeDir) {
		wg.Add(1)
		go s.CleanOld(u, threshold, &wg)
	}
	wg.Wait()
}

func (s *Service) RunTier1() {
	s.RunTier1Regenerable()
	s.RunTier1Browsers()
	s.RunTier1Updaters()
}

func (s *Service) RunTier2() {
	s.header(s.t("tier2"))
	var wg sync.WaitGroup
	wg.Add(1)
	go s.WipeContents(config.GradleCacheDir(s.HomeDir), &wg)

	cargoReg := config.CargoRegistryDir(s.HomeDir)
	if repository.IsDir(cargoReg) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cacheDir := filepath.Join(cargoReg, "cache")
			srcDir := filepath.Join(cargoReg, "src")
			before := repository.DirSize(cacheDir) + repository.DirSize(srcDir)
			target := filepath.Join(cargoReg, "cache+src")
			if s.DryRun {
				s.Log(target, s.tf("wipe_plan", model.HumanSize(before)))
				return
			}

			var failed int
			wipeEntries := func(dir string) {
				names, err := repository.Children(dir)
				if err != nil {
					return
				}
				for _, name := range names {
					if err := repository.RemoveTree(filepath.Join(dir, name)); err != nil {
						failed++
					}
				}
			}
			wipeEntries(cacheDir)
			wipeEntries(srcDir)

			after := repository.DirSize(cacheDir) + repository.DirSize(srcDir)
			freed := before - after
			if freed < 0 {
				freed = 0
			}
			s.TotalFreed.Add(freed)
			s.Failures.Add(int64(failed))
			status := s.tf("ok_index", model.HumanSize(freed))
			if failed > 0 {
				status = s.tf("ok_index_failed", model.HumanSize(freed), failed)
			}
			s.Log(target, status)
		}()
	}
	wg.Wait()
}

func (s *Service) RunTier3() {
	s.header(s.t("tier3"))
	admin := repository.IsAdmin()
	if !admin {
		s.raw(s.t("not_elevated"))
	}
	var wg sync.WaitGroup
	for _, p := range config.DefaultTier3Paths() {
		if p == config.SoftwareDistDownloadDir {
			continue
		}
		wg.Add(1)
		go s.WipeContents(p, &wg)
	}
	wg.Wait()

	dist := config.SoftwareDistDownloadDir
	if !repository.IsDir(dist) {
		return
	}
	if s.DryRun {
		s.Log(dist, s.tf("wipe_dist_plan", model.HumanSize(repository.DirSize(dist))))
	} else if admin {

		if strings.Contains(repository.ScQuery("TrustedInstaller"), "RUNNING") {
			s.Log(dist, s.t("skip_wu_busy"))
			return
		}

		wasRunning := strings.Contains(repository.ScQuery("wuauserv"), "RUNNING")
		if wasRunning {
			if err := exec.Command("net", "stop", "wuauserv").Run(); err != nil {
				s.Log(dist, s.t("skip_wu_stop"))
				return
			}
		}

		var distWg sync.WaitGroup
		distWg.Add(1)
		s.WipeContents(dist, &distWg)
		distWg.Wait()

		if wasRunning {
			if exec.Command("net", "start", "wuauserv").Run() != nil {
				s.Log("wuauserv", s.t("warn_wuauserv"))
			}
		}
	} else {
		s.Log(dist, s.t("skip_admin"))
	}
}

func (s *Service) RunExtras(paths []string) {
	s.header(s.t("extras"))
	var wg sync.WaitGroup
	threshold := s.threshold()
	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		if repository.IsProtectedPath(strings.TrimPrefix(p, "old:"), s.HomeDir) {
			s.Log(strings.TrimPrefix(p, "old:"), s.t("denied_path"))
			continue
		}
		wg.Add(1)
		if strings.HasPrefix(p, "old:") {
			go s.CleanOld(strings.TrimPrefix(p, "old:"), threshold, &wg)
			continue
		}
		if repository.IsFile(p) {
			if s.Trash {
				go s.TrashPath(p, &wg)
			} else {
				go s.RemoveFile(p, &wg)
			}
			continue
		}
		if s.Trash {
			go s.TrashDir(p, &wg)
		} else {
			go s.WipeContents(p, &wg)
		}
	}
	wg.Wait()
}
