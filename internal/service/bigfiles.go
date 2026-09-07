package service

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"clean-my-disk/internal/config"
	"clean-my-disk/internal/model"
	"clean-my-disk/internal/repository"
)

// bigFileExts adalah ekstensi installer/arsip/download yang menjadi target
// big-file hunter. Selalu lowercase.
var bigFileExts = map[string]bool{
	"exe": true, "msi": true, "msu": true, "cab": true,
	"iso": true, "img": true,
	"zip": true, "rar": true, "7z": true,
	"tar": true, "gz": true, "tgz": true, "bz2": true, "xz": true, "zst": true,
	"dmg": true, "pkg": true, "apk": true,
	// download tak selesai / menggantung
	"crdownload": true, "part": true, "download": true,
}

func (s *Service) bigFilesThreshold() time.Time {
	days := s.ThresholdDays
	if days <= 0 {
		days = model.BigFileOldDays
	}
	return time.Now().AddDate(0, 0, -days)
}

func (s *Service) bigFilesMinSize() int64 {
	if s.MinSize > 0 {
		return s.MinSize
	}
	return model.BigFileMinBytes
}

// FindBigFiles memindai roots (Downloads + Desktop) untuk file installer/
// arsip tua dan besar. Tidak menghapus apa pun. Hasil diurutkan ukuran
// terbesar dulu. Semua I/O lewat repository.
func (s *Service) FindBigFiles(home string) []model.BigFile {
	minSize := s.bigFilesMinSize()
	threshold := s.bigFilesThreshold()

	var out []model.BigFile
	for _, root := range config.BigFileRoots(home) {
		if !repository.IsDir(root) {
			continue
		}
		_ = repository.Walk(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if info.Size() < minSize {
				return nil
			}
			if !info.ModTime().Before(threshold) {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			ext = strings.TrimPrefix(ext, ".")
			if !bigFileExts[ext] {
				return nil
			}
			if s.keepMatch(p) {
				return nil
			}
			out = append(out, model.BigFile{
				Path:     p,
				Size:     info.Size(),
				Modified: info.ModTime(),
				Kind:     ext,
			})
			return nil
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Size != out[j].Size {
			return out[i].Size > out[j].Size
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// DeleteBigFiles menghapus (atau memindahkan ke Recycle Bin bila s.Trash)
// file hasil FindBigFiles. Di dry-run hanya mencatat rencana. Setiap file
// dicatat lewat Log agar progress TUI/CLI terlihat.
func (s *Service) DeleteBigFiles(files []model.BigFile) (freed int64, deleted, failed int) {
	for _, f := range files {
		sz, err := repository.FileSize(f.Path)
		if err != nil {
			if !repository.IsNotExist(err) {
				failed++
				s.Failures.Add(1)
			}
			s.Log(f.Path, s.errStatus(err))
			continue
		}
		if s.DryRun {
			s.Log(f.Path, s.tf("file_plan", model.HumanSize(sz)))
			continue
		}
		var delErr error
		if s.Trash {
			delErr = repository.MoveToTrash(f.Path)
		} else {
			delErr = repository.RemoveFile(f.Path)
		}
		if delErr != nil {
			failed++
			s.Failures.Add(1)
			s.Log(f.Path, s.t("remove_fail"))
			continue
		}
		freed += sz
		deleted++
		s.TotalFreed.Add(sz)
		s.Log(f.Path, s.tf("ok_size", model.HumanSize(sz)))
	}
	return freed, deleted, failed
}
