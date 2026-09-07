package repository

import (
	"fmt"
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
	return exec.Command("schtasks", "/Create", "/TN", name, "/TR", `"`+executable+`" --all --notify`,
		"/SC", "WEEKLY", "/D", "SUN", "/ST", "03:00", "/F").Run()
}

func RemoveTask(name string) error {
	return exec.Command("schtasks", "/Delete", "/TN", name, "/F").Run()
}

func ScQuery(serviceName string) string {
	out, _ := exec.Command("sc", "query", serviceName).CombinedOutput()
	return string(out)
}

// ToastAppID adalah Application User Model ID yang dipakai untuk notifikasi.
// Agar toast tampil walau app tidak sedang berjalan (mis. dari Task Scheduler),
// ID ini juga ditulis ke properti System.AppUserModel.ID shortcut di Start Menu
// via EnsureToastRegistration.
const ToastAppID = "CleanMyDisk"

// runPS menjalankan skrip PowerShell hidden dan mengembalikan error berisi
// output bila gagal (bukan hanya exit code) agar diagnosis mudah.
func runPS(script string) error {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script).CombinedOutput()
	if err == nil {
		return nil
	}
	if msg := strings.TrimSpace(string(out)); msg != "" {
		return fmt.Errorf("%v: %s", err, msg)
	}
	return err
}

// EnsureToastRegistration membuat (atau memperbaiki) shortcut clean-my-disk di
// Start Menu user dengan AppUserModelID = ToastAppID. Windows hanya menampilkan
// toast untuk app unpackaged bila shortcut ber-AUMID cocok ada di Start Menu.
// Idempoten: bila shortcut sudah menunjuk ke executable ini, tidak dilakukan
// apa-apa (dan tidak ada kompilasi Add-Type).
func EnsureToastRegistration(executable string) error {
	psQuote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	lnkName := "clean-my-disk.lnk"
	lines := []string{
		"$ErrorActionPreference = 'Stop'",
		"$lnk = Join-Path ([Environment]::GetFolderPath('Programs')) " + psQuote(lnkName),
		"$need = $true",
		"if (Test-Path -LiteralPath $lnk) {",
		"  $sh = New-Object -ComObject WScript.Shell",
		"  $sc = $sh.CreateShortcut($lnk)",
		"  $tgt = $sc.TargetPath",
		"  $aumid = ''",
		"  try {",
		"    $sh2 = New-Object -ComObject Shell.Application",
		"    $ldir = Split-Path -Parent $lnk",
		"    $it = $sh2.Namespace($ldir).ParseName((Split-Path -Leaf $lnk))",
		"    $aumid = [string]$it.ExtendedProperty('System.AppUserModel.ID')",
		"  } catch { $aumid = '' }",
		"  if ($tgt -ieq " + psQuote(executable) + " -and $aumid -eq " + psQuote(ToastAppID) + ") { $need = $false }",
		"}",
		"if ($need) {",
		"  $sh = New-Object -ComObject WScript.Shell",
		"  $sc = $sh.CreateShortcut($lnk)",
		"  $sc.TargetPath = " + psQuote(executable),
		"  $sc.WorkingDirectory = " + psQuote(filepath.Dir(executable)),
		"  $sc.IconLocation = " + psQuote(executable+",0"),
		"  $sc.Description = 'Windows Cache & Junk Cleaner'",
		"  $sc.Save()",
		"  $src = @'" + toastRegCS + "'@",
		"  Add-Type -TypeDefinition $src",
		"  [LnkAumid]::Set($lnk, " + psQuote(ToastAppID) + ")",
		"}",
	}
	ps := strings.Join(lines, "\n")
	return runPS(ps)
}

// ShowToast menampilkan notifikasi Windows toast memakai ToastAppID. Sebelum
// dipanggil, EnsureToastRegistration harus sudah dijalankan agar toast tampil
// walau app tidak aktif/foreground.
func ShowToast(title, message string) error {
	psQuote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	ps := "[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null\n" +
		"$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)\n" +
		"$textNodes = $template.GetElementsByTagName('text')\n" +
		"$textNodes.Item(0).AppendChild($template.CreateTextNode(" + psQuote(title) + ")) | Out-Null\n" +
		"$textNodes.Item(1).AppendChild($template.CreateTextNode(" + psQuote(message) + ")) | Out-Null\n" +
		"$toast = [Windows.UI.Notifications.ToastNotification]::new($template)\n" +
		"[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier(" + psQuote(ToastAppID) + ").Show($toast) | Out-Null\n"
	return runPS(ps)
}

// toastRegCS berisi helper C# untuk menulis System.AppUserModel.ID ke file
// .lnk lewat IPropertyStore shell32 (WScript.Shell tidak bisa melakukannya).
// Dikompilasi on-the-fly oleh Add-Type hanya saat shortcut perlu dibuat.
var toastRegCS = `
using System;
using System.Runtime.InteropServices;

public static class LnkAumid
{
    [ComImport]
    [Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IPropertyStore
    {
        int GetCount(out uint cProps);
        int GetAt(uint iProp, out PROPERTYKEY pkey);
        int GetValue(ref PROPERTYKEY key, out PROPVARIANT pv);
        int SetValue(ref PROPERTYKEY key, ref PROPVARIANT pv);
        int Commit();
    }

    [StructLayout(LayoutKind.Sequential, Pack = 4)]
    struct PROPERTYKEY
    {
        public Guid fmtid;
        public uint pid;
    }

    [StructLayout(LayoutKind.Sequential)]
    struct PROPVARIANT
    {
        public ushort vt;
        public ushort wReserved1;
        public ushort wReserved2;
        public ushort wReserved3;
        public IntPtr data;
    }

    const ushort VT_LPWSTR = 31;
    const uint STGM_READWRITE = 2;

    [DllImport("shell32.dll", CharSet = CharSet.Unicode)]
    static extern int SHGetPropertyStoreFromParsingName(string pszPath, IntPtr pbc, uint flags, ref Guid riid, out IPropertyStore ppStore);

    static readonly Guid IID_IPropertyStore = new Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99");
    static readonly PROPERTYKEY PKEY_AppUserModel_ID = new PROPERTYKEY
    {
        fmtid = new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"),
        pid = 5
    };

    public static void Set(string lnkPath, string appId)
    {
        IPropertyStore store;
        Guid iid = IID_IPropertyStore;
        if (SHGetPropertyStoreFromParsingName(lnkPath, IntPtr.Zero, STGM_READWRITE, ref iid, out store) != 0)
            throw new InvalidOperationException("SHGetPropertyStoreFromParsingName failed");
        try
        {
            PROPVARIANT pv = new PROPVARIANT();
            pv.vt = VT_LPWSTR;
            pv.data = Marshal.StringToCoTaskMemUni(appId);
            try
            {
                PROPERTYKEY k = PKEY_AppUserModel_ID;
                store.SetValue(ref k, ref pv);
                store.Commit();
            }
            finally
            {
                Marshal.FreeCoTaskMem(pv.data);
            }
        }
        finally
        {
            Marshal.ReleaseComObject(store);
        }
    }
}
`
