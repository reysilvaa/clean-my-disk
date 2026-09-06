package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Home         string
	ExtraPaths   []string
	KeepPatterns []string
}

func Home() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

func Load(home string) *Config {
	file := filepath.Join(home, ".clean-caches.paths")
	return &Config{
		Home:         home,
		ExtraPaths:   LoadExtraPaths(file),
		KeepPatterns: LoadKeepPatterns(file),
	}
}

var BrowserCacheTargets = []string{
	"Cache",
	"Code Cache",
	"GPUCache",
	"DawnCache",
	"DawnGraphiteCache",
	"DawnWebGPUCache",
	"ShaderCache",
	"GrShaderCache",
	"Service Worker/CacheStorage",
	"Service Worker/ScriptCache",
	"Media Cache",
	"jump_list_icons",
}

var FirefoxCacheTargets = []string{
	"cache2",
	"OfflineCache",
	"startupCache",
	"Cache",
}

func DefaultTier1Paths(home string) []string {
	return []string{
		filepath.Join(home, "AppData/Local/Temp"),
		filepath.Join(home, "AppData/Local/CrashDumps"),
		filepath.Join(home, "AppData/Local/Microsoft/Windows/INetCache"),
		filepath.Join(home, "AppData/Local/Microsoft/Windows/WER/ReportArchive"),
		filepath.Join(home, "AppData/Local/Microsoft/Windows/WER/ReportQueue"),
		filepath.Join(home, "AppData/Local/Microsoft/Windows/WER/Temp"),
		filepath.Join(home, "AppData/Local/NVIDIA/DXCache"),
		filepath.Join(home, "AppData/Local/NVIDIA/GLCache"),
		filepath.Join(home, "AppData/Local/AMD/DxCache"),
		filepath.Join(home, "AppData/Local/AMD/GLCache"),
		filepath.Join(home, "AppData/Local/Intel/ShaderCache"),
		filepath.Join(home, "AppData/Local/D3DSCache"),
		filepath.Join(home, ".npm/_cacache"),
		filepath.Join(home, "AppData/Local/npm-cache"),
		filepath.Join(home, "AppData/Local/pnpm-cache"),
		filepath.Join(home, ".bun/install/cache"),
		filepath.Join(home, "AppData/Local/go-build"),
		filepath.Join(home, "AppData/Local/gopls"),
		filepath.Join(home, "AppData/Local/goimports"),
		filepath.Join(home, "AppData/Local/ms-playwright"),
		filepath.Join(home, ".cache/puppeteer"),
		filepath.Join(home, ".cache/chrome-devtools-mcp"),
		filepath.Join(home, "AppData/Local/node-gyp/Cache"),
		filepath.Join(home, "AppData/Local/Yarn/Cache"),
		filepath.Join(home, "AppData/Local/pip/cache"),
		filepath.Join(home, "AppData/Local/uv/cache"),
		filepath.Join(home, "AppData/Local/pypoetry/cache"),
		filepath.Join(home, "AppData/Local/Composer/cache"),
		filepath.Join(home, "AppData/Local/Cypress/Cache"),
		filepath.Join(home, "AppData/Local/rustup/downloads"),
		filepath.Join(home, "AppData/Local/SquirrelTemp"),
		filepath.Join(home, "AppData/Local/Spotify/Storage"),
		filepath.Join(home, ".android/cache"),
		filepath.Join(home, ".android/build-cache"),
		filepath.Join(home, "AppData/Local/javasharedresources"),
		filepath.Join(home, ".wdm"),
		filepath.Join(home, "AppData/Roaming/Code/Cache"),
		filepath.Join(home, "AppData/Roaming/Code/CachedData"),
		filepath.Join(home, "AppData/Roaming/Code/CachedExtensionVSIXs"),
		filepath.Join(home, "AppData/Roaming/Code/Code Cache"),
		filepath.Join(home, "AppData/Roaming/Code/GPUCache"),
		filepath.Join(home, "AppData/Roaming/Antigravity IDE/Cache"),
		filepath.Join(home, "AppData/Roaming/Antigravity IDE/CachedData"),
		filepath.Join(home, "AppData/Roaming/Antigravity IDE/CachedExtensionVSIXs"),
		filepath.Join(home, "AppData/Roaming/Antigravity IDE/GPUCache"),
		filepath.Join(home, "AppData/Roaming/Cursor/Cache"),
		filepath.Join(home, "AppData/Roaming/Cursor/CachedData"),
		filepath.Join(home, "AppData/Roaming/Cursor/CachedExtensionVSIXs"),
		filepath.Join(home, "AppData/Roaming/Cursor/GPUCache"),
		filepath.Join(home, "AppData/Roaming/Windsurf/Cache"),
		filepath.Join(home, "AppData/Roaming/Windsurf/CachedData"),
		filepath.Join(home, "AppData/Roaming/Windsurf/CachedExtensionVSIXs"),
		filepath.Join(home, "AppData/Roaming/Windsurf/GPUCache"),
		filepath.Join(home, "AppData/Roaming/OpenCode/Cache"),
		filepath.Join(home, "AppData/Roaming/OpenCode/CachedData"),
		filepath.Join(home, "AppData/Roaming/OpenCode/GPUCache"),
	}
}

func DefaultUpdaterPaths(home string) []string {
	return []string{
		filepath.Join(home, "AppData/Local/vortex-updater"),
		filepath.Join(home, "AppData/Local/podman-desktop-updater"),
		filepath.Join(home, "AppData/Local/canva-updater"),
		filepath.Join(home, "AppData/Local/ichigo-unlocker-updater"),
		filepath.Join(home, "AppData/Local/antigravity-updater"),
		filepath.Join(home, "AppData/Local/@codebufffreebuff-desktop-updater"),
		filepath.Join(home, "AppData/Local/Discord/packages"),
		filepath.Join(home, "AppData/Local/Slack/packages"),
		filepath.Join(home, "AppData/Local/Spotify/Update"),
		filepath.Join(home, "AppData/Local/Postman-updater"),
		filepath.Join(home, "AppData/Local/Notion-updater"),
		filepath.Join(home, "AppData/Local/Figma-updater"),
		filepath.Join(home, "AppData/Local/WhatsApp/packages"),
		filepath.Join(home, "AppData/Local/VS Revo Group/Revo Uninstaller Pro/RegBackup"),
	}
}

func BrowserUserDirs(home string) map[string]string {
	return map[string]string{
		"Chrome":   filepath.Join(home, "AppData/Local/Google/Chrome/User Data"),
		"Edge":     filepath.Join(home, "AppData/Local/Microsoft/Edge/User Data"),
		"Brave":    filepath.Join(home, "AppData/Local/BraveSoftware/Brave-Browser/User Data"),
		"Firefox":  filepath.Join(home, "AppData/Roaming/Mozilla/Firefox"),
		"Opera":    filepath.Join(home, "AppData/Roaming/Opera Software/Opera Stable"),
		"Opera GX": filepath.Join(home, "AppData/Roaming/Opera Software/Opera GX Stable"),
		"Vivaldi":  filepath.Join(home, "AppData/Local/Vivaldi/User Data"),
		"Arc":      filepath.Join(home, "AppData/Local/Arc/User Data"),
		"Zen":      filepath.Join(home, "AppData/Roaming/zen/Profiles"),
		"Chromium": filepath.Join(home, "AppData/Local/Chromium/User Data"),
	}
}

func GradleCacheDir(home string) string {
	return filepath.Join(home, ".gradle/caches")
}

func CargoRegistryDir(home string) string {
	return filepath.Join(home, ".cargo/registry")
}

const (
	WindowsTempDir          = `C:\Windows\Temp`
	NvidiaDownloaderDir     = `C:\ProgramData\NVIDIA Corporation\Downloader`
	SoftwareDistDownloadDir = `C:\Windows\SoftwareDistribution\Download`
	WindowsLogsCBSDir       = `C:\Windows\Logs\CBS`
	WindowsMinidumpDir      = `C:\Windows\Minidump`
	ProgramDataWERArchive   = `C:\ProgramData\Microsoft\Windows\WER\ReportArchive`
	ProgramDataWERQueue     = `C:\ProgramData\Microsoft\Windows\WER\ReportQueue`
	ProgramDataWERTemp      = `C:\ProgramData\Microsoft\Windows\WER\Temp`
)

func DefaultTier2Paths(home string) []string {
	cargoReg := CargoRegistryDir(home)
	return []string{
		GradleCacheDir(home),
		filepath.Join(cargoReg, "cache"),
		filepath.Join(cargoReg, "src"),
	}
}

func DefaultTier3Paths() []string {
	return []string{
		WindowsTempDir,
		NvidiaDownloaderDir,
		SoftwareDistDownloadDir,
		WindowsLogsCBSDir,
		WindowsMinidumpDir,
		ProgramDataWERArchive,
		ProgramDataWERQueue,
		ProgramDataWERTemp,
	}
}

func LoadExtraPaths(file string) []string {
	var out []string
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	home := Home()
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		prefix := ""
		if strings.HasPrefix(line, "old:") {
			prefix = "old:"
			line = strings.TrimPrefix(line, "old:")
		}

		line = os.ExpandEnv(line)

		if home != "" {
			if line == "~" {
				line = home
			} else if strings.HasPrefix(line, "~/") || strings.HasPrefix(line, `~\`) {
				line = filepath.Join(home, line[2:])
			}
		}

		out = append(out, prefix+line)
	}
	return out
}

func LoadKeepPatterns(file string) []string {
	var out []string
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if p, ok := strings.CutPrefix(line, "keep:"); ok {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}
