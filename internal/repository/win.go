package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"clean-my-disk/internal/model"

	"golang.org/x/sys/windows"
)

func FixedDrives() []model.Drive {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}

	var out []model.Drive
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		rootPtr, err := windows.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		if windows.GetDriveType(rootPtr) != windows.DRIVE_FIXED {
			continue
		}
		var freeAvail, totalBytes, totalFree uint64
		if err := windows.GetDiskFreeSpaceEx(rootPtr, &freeAvail, &totalBytes, &totalFree); err != nil {
			continue
		}
		out = append(out, model.Drive{
			Letter: string(rune('A' + i)),
			Free:   int64(freeAvail),
			Total:  int64(totalBytes),
		})
	}
	return out
}

func GetDriveSpace() (free, total int64, percentUsed float64) {
	return model.AggregateDrives(FixedDrives())
}

func IsAdmin() bool {
	return exec.Command("net", "session").Run() == nil
}

func IsProtectedPath(p, home string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		return true
	}
	abs = filepath.Clean(abs)
	sep := string(filepath.Separator)

	if vol := filepath.VolumeName(abs); vol != "" && strings.EqualFold(abs, vol+sep) {
		return true
	}

	for _, env := range []string{"SystemRoot", "ProgramFiles", "ProgramFiles(x86)", "ProgramData"} {
		if root := os.Getenv(env); root != "" && isWithin(abs, filepath.Clean(root)) {
			return true
		}
	}

	if home != "" {
		for cur := filepath.Clean(home); cur != "" && len(cur) > len(filepath.VolumeName(cur))+1; cur = filepath.Dir(cur) {
			if strings.EqualFold(abs, cur) {
				return true
			}
		}
	}
	return false
}

func isWithin(p, root string) bool {
	sep := string(filepath.Separator)
	return strings.EqualFold(p, root) || strings.HasPrefix(strings.ToLower(p), strings.ToLower(root)+sep)
}

func RunElevated(executable string, args []string) error {
	psQuote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	ps := "Start-Process -Verb RunAs -FilePath " + psQuote(executable)
	if len(args) > 0 {
		quoted := make([]string, len(args))
		for i, a := range args {
			quoted[i] = psQuote(a)
		}
		ps += " -ArgumentList " + strings.Join(quoted, ",")
	}
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
}

func MoveToTrash(path string) error {
	psQuote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	ps := "Add-Type -AssemblyName Microsoft.VisualBasic; " +
		"if (Test-Path -LiteralPath " + psQuote(path) + " -PathType Container) { " +
		"[Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory(" + psQuote(path) + ",'OnlyErrorDialogs','SendToRecycleBin') } else { " +
		"[Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile(" + psQuote(path) + ",'OnlyErrorDialogs','SendToRecycleBin') }"
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
}

func EmptyRecycleBin() error {
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Clear-RecycleBin -Force -ErrorAction SilentlyContinue").Run()
}

func RegisterWeeklyTask(name, executable string) error {
	return exec.Command("schtasks", "/Create", "/TN", name, "/TR", `"`+executable+`" --all`,
		"/SC", "WEEKLY", "/D", "SUN", "/ST", "03:00", "/F").Run()
}

func RemoveTask(name string) error {
	return exec.Command("schtasks", "/Delete", "/TN", name, "/F").Run()
}

func ScQuery(serviceName string) string {
	out, _ := exec.Command("sc", "query", serviceName).CombinedOutput()
	return string(out)
}
