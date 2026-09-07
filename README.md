# clean-my-disk

Windows Cache & Junk Cleaner — CLI + TUI interaktif, ditulis dalam Go.
Membersihkan cache regenerable, cache browser, junk updater, cache dev, dan file sistem
(butuh admin) — dengan proteksi berlapis dan laporan ruang yang dibebaskan.

**Versi saat ini: v1.6.0**

## Fitur

- **3 tier pembersihan** sesuai tingkat risiko, plus path ekstra non-hardcode.
- **TUI interaktif** (bubbletea) saat dijalankan tanpa argumen: pilih target, toggle dry-run/trash, estimasi ukuran per item.
- **CLI** penuh untuk skrip & automasi: `--dry-run`, `--scan`, `--json`, exit code `0/1/2`.
- **Bilingual** output id/en (`--lang en`).
- **Laporan akurat**: ruang dibebaskan diukur per target (delta sebelum/sesudah), total + kegagalan terhitung nyata.
- **Info disk per-drive** (bukan agregat yang menyesatkan).
- **Proteksi path**: whitelist deny untuk akar drive & direktori sistem; dukungan `keep:` glob di config.
- **Jadwal mingguan** via Task Scheduler (`--install-schedule`) — berjalan `--all` dan menampilkan **notifikasi toast Windows** berisi hasil (`--notify`).
- **Big-file hunter** (`--big-files`): temukan installer/arsip besar & tua di Downloads/Desktop dengan pratinjau sebelum dihapus.

## Tier

| Tier | Isi | Butuh admin? | Flag |
|---|---|---|---|
| 1 | Cache regenerable (Temp, CrashDumps, npm/pip/uv, shader), cache browser (Chrome/Edge/Brave/Firefox/Opera/Vivaldi/Arc/Zen/Chromium), junk updater > 14 hari | Tidak | default |
| 2 | Dev cache (Gradle, Cargo `cache`+`src` — index dipertahankan) | Tidak | `--dev` |
| 3 | Sistem: `C:\Windows\Temp`, NVIDIA Downloader, SoftwareDistribution (stop/start wuauserv), WER, Minidump | Ya | `--admin` |
| Ekstra | Path non-hardcode via `--path` atau config | — | `--path` |
| Recycle Bin | Kosongkan Recycle Bin | — | `--recycle` |

Tidak pernah disentuh: pnpm store, WSL vhdx, WinSxS, profil browser, data login, DB.

## Instalasi & build

```bash
# butuh Go 1.22+; hanya berjalan di Windows
go build -o clean-my-disk.exe .
go test -count=1 ./...
go vet ./...
```

Override versi saat rilis:

```bash
go build -ldflags "-X clean-my-disk/internal/model.Version=1.5.0" -o clean-my-disk.exe .
```

## Pemakaian

```bash
clean-my-disk.exe                    # TUI interaktif
clean-my-disk.exe --dry-run          # simulasi Tier 1 (aman, tidak menghapus)
clean-my-disk.exe --all              # Tier 1 + 2 + 3
clean-my-disk.exe --admin            # Tier 3; auto-elevasi UAC bila belum admin
clean-my-disk.exe --scan             # laporan ukuran semua target, tanpa hapus
clean-my-disk.exe --scan --json      # sama, format JSON
clean-my-disk.exe --dry-run --json   # rencana run dalam JSON
clean-my-disk.exe --notify           # daftarkan AUMID (otomatis) + tampilkan toast hasil setelah selesai
clean-my-disk.exe --version

### Big files (installer/arsip besar & tua)

```bash
clean-my-disk.exe --big-files                 # preview + konfirmasi interaktif per file
clean-my-disk.exe --big-files --dry-run       # preview saja, tanpa hapus
clean-my-disk.exe --big-files --scan --json   # daftar kandidat dalam JSON
clean-my-disk.exe --trash --big-files         # pindah ke Recycle Bin, bukan hapus permanen
```

Memindai `Downloads` dan `Desktop` user saat ini untuk file ber-ekstensi
installer/arsip (`.exe`, `.msi`, `.iso`, `.zip`, `.rar`, `.7z`, `.tar.gz`, ...)
berukuran **≥ 100 MB** dan berumur **> 30 hari** (atur lewat `--min-size` MB
dan `--days`). `keep:` di config tetap berlaku.

Di TUI: centang item **Big Files** lalu tekan Enter → muncul layar pilih
per-file (space = centang, `a` = semua, `n` = kosongkan); hanya file yang
dicentang yang dihapus saat eksekusi.

Saat stdin berupa terminal, `--big-files` menampilkan tabel bernomor lalu
menanyakan file mana yang dihapus (`a` = semua, nomor dipisah spasi/koma,
Enter = batal). Tanpa terminal (pipa/cron) semua kandidat dihapus langsung
seperti tier — gunakan `--dry-run` untuk pratinjau atau `--trash` agar
file masuk Recycle Bin.
```

### Path ekstra

```bash
clean-my-disk.exe --path "D:\Downloads\Sampah"                 # wipe isi folder
clean-my-disk.exe --path "old:D:\Backup\Lama"                  # hapus file > 14 hari saja
clean-my-disk.exe --path "C:\cache\satu.dll"                   # hapus file tunggal
clean-my-disk.exe --trash --path "D:\x"                        # ke Recycle Bin, bukan permanen
clean-my-disk.exe --path "D:\a" --path "D:\b"                  # bisa diulang
```

Config permanen `~/.clean-caches.paths` (satu path per baris, `#` komentar,
`%ENV%` dan `~` diekspansi):

```ini
# dibaca otomatis setiap run
D:\Downloads\Sampah
old:D:\Backup\Lama
keep:*.exe            # lindungi file bermatch di dalam target
keep:installer*.msi
```

### Proteksi deny-path (bawaan, tak bisa dilewati)

Path berikut **ditolak keras** sebagai `--path`/config — tidak akan dihapus walau
sengaja ditulis, termasuk mode `--trash`:

- Akar drive mana pun (`C:\`, `D:\`, ...)
- `%SystemRoot%` & seluruh subtree-nya (`C:\Windows\...`)
- `Program Files`, `Program Files (x86)` & subtree-nya
- `ProgramData` & subtree-nya
- Direktori home user dan leluhurnya sampai akar drive (profil & induk profil)

Path di dalam profil sendiri (mis. `C:\Users\Me\AppData\Local\Temp`) tetap diterima.
Catatan: profil *pengguna lain* (`C:\Users\Lain\...`) tidak otomatis dilindungi —
jalankan `--dry-run` dulu sebelum percaya pada daftar path ekstra.

### Exit code

| Kode | Arti |
|---|---|
| 0 | Bersih, atau selesai tanpa kegagalan |
| 1 | Ada item gagal/terkunci (lihat laporan) |
| 2 | Error argumen/flag atau fatal |

## Jadwal (Task Scheduler)

```bash
clean-my-disk.exe --install-schedule      # Minggu 03:00, menjalankan --all + notifikasi toast
clean-my-disk.exe --uninstall-schedule
```

## Arsitektur

```
main.go                          entry tipis (3 baris)
internal/
  model/         entities & konstanta — tanpa dependensi
  config/        target default per tier + config user + deteksi home
  repository/    SATU-SATUNYA akses OS/disk: fs, drive, admin, trash, recycle, schtasks, deny-path
  service/       use-case: cleaning (tier, extras), scanning, scheduling, elevasi
  view/          presentasi: CLI renderer (teks + JSON) + TUI (bubbletea)
  controller/    entry CLI: flag → service → view; exit code 0/1/2
  i18n/          tabel string id/en
```

Dependensi satu arah ke bawah (`controller → view/service → repository/config/model`),
tidak ada cycle. Service tidak pernah memanggil `os.*` langsung — semua I/O lewat
repository agar bisa diuji. Tidak ada interface buatan sebelum ada implementasi kedua.

## Pengujian

```bash
go test -count=1 ./...
```

Mencakup: akuntansi freed-space yang akurat, keamanan junction (tidak menembus target),
deny-path proteksi, parsing flag & tier selection, model TUI (toggle, seleksi, render),
HumanSize, config parser, filter & akuntansi big-file hunter (umur/ukuran/ekstensi,
keep-pattern, dry-run), dan parsing seleksi interaktif.

## Batasan & catatan jujur

- Windows-only (memakai `golang.org/x/sys/windows`, schtasks, PowerShell, net session).
- Notifikasi toast memakai AUMID `CleanMyDisk` yang **didaftarkan** lewat
  `clean-my-disk.lnk` di `%APPDATA%\...\Start Menu\Programs` (properti
  `System.AppUserModel.ID` ditulis via `IPropertyStore` shell32). Registrasi
  terjadi otomatis saat `--notify` pertama kali atau `--install-schedule`;
  idempoten dan self-healing — target exe yang berubah atau AUMID yang hilang
  diperbaiki sendiri (helper C# dikompilasi on-the-fly hanya saat perlu). Butuh
  Windows 10+ dan hak tulis ke Start Menu user; bila gagal hanya dicatat, tidak
  mengubah exit code.
- `--admin` memicu prompt UAC via `Start-Process -Verb RunAs`; bila dibatalkan, program
  keluar dengan kode 1 — jalankan dari shell administrator.
- Recycle Bin diukur lewat delta ruang bebas semua drive fixed.
- File yang terkunci (dipakai proses berjalan) dilewati dan dihitung sebagai kegagalan — tidak merusak.
