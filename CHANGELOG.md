# Changelog

## v1.5.0 (2026-09-07)

### Keamanan
- **Deny-path proteksi**: akar drive, `%SystemRoot%`, `Program Files`, `ProgramData`, dan home user ditolak keras sebagai `--path`/config — termasuk mode `--trash` dan `old:`.
- **`--admin` auto-elevasi UAC** (`Start-Process -Verb RunAs`) bila belum elevated; `--dry-run`/`--scan` tidak memicu prompt.
- **`MoveToTrash`**: escape kutip tunggal path (mencegah injection/kerusakan path ber-`'`).

### Perbaikan laporan & error handling
- Info disk **per-drive** (bukan agregat yang menyesatkan).
- Ikon status jujur: `✗` untuk baris gagal/terkunci/ditolak, `✓` hanya `OK`, `·` netral — bukan `✓` untuk semua.
- Error yang tadinya diam-diam kini dilaporkan: `CleanOld` walk gagal, `EmptyRecycleBin` gagal (kunci i18n `recycle_fail`).
- `--scan` **dedupe** path yang muncul sebagai target default sekaligus baris config.
- Divider Recycle Bin tidak lagi merusak output `--json`.

### Arsitektur
- Service bebas `fmt.Println`: semua output lewat `Out io.Writer`/`OnLog` (header via `header()`/`raw()`), stdout tidak diakses langsung oleh layer service.

### Target baru (Tier 1)
- `gopls` & `goimports` cache (analisis Go LSP).
- Browser-download cache: `.cache/puppeteer`, `.cache/chrome-devtools-mcp`, `AppData/Local/ms-playwright` (Playwright Windows).

### Lainnya
- Flag `--version` + `-ldflags -X` untuk set versi saat rilis.
- README.md lengkap (fitur, tier, contoh, deny-path, arsitektur, batasan).

### Pengujian
- Test baru: deny-path (`IsProtectedPath`), wiring tier (gopls/goimports/browser-download), output via `Out`, error-path, `parseFlags --version`.
- `TestNewModel` kini terisolasi dari config home nyata (deterministik lintas mesin).
