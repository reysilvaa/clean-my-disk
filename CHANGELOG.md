# Changelog

## v1.6.0 (2026-09-07)

### Fitur baru
- **Big-file hunter (`--big-files`)**: memindai `Downloads` & `Desktop` untuk
  installer/arsip besar & tua (`.exe/.msi/.iso/.zip/...` ≥ 100 MB, > 30 hari;
  atur via `--min-size`/`--days`; hormati `keep:`). Preview tabel bernomor lalu
  konfirmasi interaktif per file (`a` = semua, nomor, Enter = batal). Tanpa
  terminal: semua kandidat dihapus tier-style — pratinjau via `--dry-run`,
  amankan via `--trash`. Output JSON via `--big-files --json`.
- **Notifikasi toast Windows (`--notify`)**: menampilkan hasil pembersihan
  (ruang dibebaskan + jumlah gagal) via PowerShell `ToastNotificationManager`
  setelah run. Jadwal mingguan kini terdaftar sebagai `--all --notify`.
- **Registrasi AUMID otomatis**: shortcut `clean-my-disk.lnk` dibuat/diperbaiki
  di `Start Menu\Programs` dengan `System.AppUserModel.ID = CleanMyDisk`
  (IPropertyStore shell32; helper C# dikompilasi `Add-Type` hanya saat perlu)
  sehingga toast tampil walau app tidak berjalan (kasus Task Scheduler).
  Registrasi idempoten & self-healing (target/AUMID diperiksa tiap kali,
  diperbaiki bila berubah/hilang): dipicu `--notify` & `--install-schedule`.
- PowerShell runner repo memakai `CombinedOutput` — error registrasi/toast
  dilaporkan berisi penyebab, bukan sekadar exit code.

### TUI
- Target baru **Big Files: Installer/Arsip** (opsional, default nonaktif) —
  internal digeneralisasi: tiap item punya runner sendiri (bukan indeks
  hardcoded), badge risiko & estimasi per item dari data item.
- **Pilih-per-file di TUI**: mencentang Big Files lalu Enter membuka layar
  pemilihan kandidat (dari scan awal; toggle per file, `a` = semua, `n` =
  kosong, estimasi terpilih live) — hanya file yang dicentang yang dihapus,
  mode live/dry-run & trash tetap dihormati. Kandidat yang muncul setelah scan
  tidak dihapus tanpa pratinjau (aman). Jalur CLI `--big-files` tidak berubah.

### Lainnya
- `bigfiles.go`: `FindBigFiles` (filter ukuran/umur/ekstensi/keep, urut
  menurun) + `DeleteBigFiles` (akuntansi freed/failed, dry-run & trash-aware,
  log per file).
- Model: tipe `BigFile`/`BigFileReport`, konstanta default (30 hari, 100 MB).

### Pengujian
- Filter & urutan FindBigFiles (umur, ukuran, ekstensi, direktori, root
  hilang), keep-pattern, akuntansi DeleteBigFiles + dry-run tanpa efek,
  parse seleksi interaktif, flag default `--big-files`/`--notify`, item TUI
  baru.

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
