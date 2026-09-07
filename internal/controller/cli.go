package controller

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"clean-my-disk/internal/config"
	"clean-my-disk/internal/i18n"
	"clean-my-disk/internal/model"
	"clean-my-disk/internal/service"
	"clean-my-disk/internal/view"

	tea "github.com/charmbracelet/bubbletea"
)

type pathList []string

func (p *pathList) String() string { return strings.Join(*p, ", ") }
func (p *pathList) Set(v string) error {
	*p = append(*p, v)
	return nil
}

type options struct {
	extra          pathList
	version        bool
	dryRun         bool
	dev            bool
	admin          bool
	recycle        bool
	all            bool
	cli            bool
	scan           bool
	json           bool
	trash          bool
	bigFiles       bool
	notify         bool
	installSched   bool
	uninstallSched bool
	minSizeMB      int64
	days           int
	lang           string
}

func registerFlags(fs *flag.FlagSet, o *options) {
	fs.Var(&o.extra, "path", "direktori/file ekstra non-hardcode (bisa diulang; prefix 'old:' = hanya file > 14 hari)")
	fs.BoolVar(&o.version, "version", false, "tampilkan versi")
	fs.BoolVar(&o.dryRun, "dry-run", false, "tampilkan rencana & estimasi ukuran, tanpa menghapus")
	fs.BoolVar(&o.dev, "dev", false, "Tier 2: dev cache (re-download)")
	fs.BoolVar(&o.admin, "admin", false, "Tier 3: sistem (butuh admin; auto-elevasi UAC bila perlu)")
	fs.BoolVar(&o.recycle, "recycle", false, "kosongkan Recycle Bin")
	fs.BoolVar(&o.all, "all", false, "Tier 1 + 2 + 3 (tanpa Recycle Bin)")
	fs.BoolVar(&o.cli, "cli", false, "paksa mode CLI langsung tanpa TUI")
	fs.BoolVar(&o.scan, "scan", false, "laporan ukuran semua target, tanpa menghapus apa pun")
	fs.BoolVar(&o.json, "json", false, "output JSON (untuk automasi)")
	fs.BoolVar(&o.trash, "trash", false, "path ekstra dipindah ke Recycle Bin, bukan dihapus permanen")
	fs.BoolVar(&o.bigFiles, "big-files", false, "cari file installer/arsip besar & tua di Downloads/Desktop (>=100 MB, >30 hari; preview + konfirmasi sebelum hapus)")
	fs.BoolVar(&o.notify, "notify", false, "tampilkan notifikasi Windows toast berisi hasil setelah selesai")
	fs.BoolVar(&o.installSched, "install-schedule", false, "jadwalkan pembersihan mingguan (Task Scheduler, Minggu 03:00, dengan notifikasi)")
	fs.BoolVar(&o.uninstallSched, "uninstall-schedule", false, "hapus jadwal Task Scheduler")
	fs.Int64Var(&o.minSizeMB, "min-size", 0, "lewati target di bawah ukuran ini (MB, 0 = nonaktif)")
	fs.IntVar(&o.days, "days", 0, "umur file lama dalam hari (0 = default 14)")
	fs.StringVar(&o.lang, "lang", "id", "bahasa output: id | en")
}

func printUsage(out io.Writer) {
	fmt.Fprintf(out, "clean-my-disk v%s — Windows Cache & Junk Cleaner\n\n", model.Version)
	fmt.Fprintln(out, "Penggunaan clean-my-disk:")
	fmt.Fprintln(out, "  clean-my-disk.exe [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	tmp := flag.NewFlagSet("clean-my-disk", flag.ContinueOnError)
	tmp.SetOutput(out)
	registerFlags(tmp, &options{})
	tmp.PrintDefaults()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Contoh:")
	fmt.Fprintln(out, "  clean-my-disk.exe                    # Buka menu interaktif TUI")
	fmt.Fprintln(out, "  clean-my-disk.exe --dry-run          # Simulasi pembersihan Tier 1")
	fmt.Fprintln(out, "  clean-my-disk.exe --all              # Bersihkan Tier 1 + 2 + 3 via CLI")
	fmt.Fprintln(out, "  clean-my-disk.exe --scan             # Laporan target tanpa hapus")
	fmt.Fprintln(out, "  clean-my-disk.exe --big-files        # Preview + konfirmasi hapus file besar tua")
	fmt.Fprintln(out, "  clean-my-disk.exe --path \"C:\\cache\"  # Tambah path kustom")
	fmt.Fprintln(out, "  clean-my-disk.exe --trash --path \"D:\\x\" # Ekstra ke Recycle Bin")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Config file ~/.clean-caches.paths: satu path per baris; '#' = komentar;")
	fmt.Fprintln(out, "  prefix 'old:' = hapus file lama saja; 'keep:<glob>' = lindungi file.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Proteksi bawaan: akar drive, Windows, Program Files, ProgramData,")
	fmt.Fprintln(out, "  dan profil user tidak pernah diterima sebagai --path/config.")
}

func parseFlags(args []string) (*options, error) {
	o := &options{}
	fs := flag.NewFlagSet("clean-my-disk", flag.ContinueOnError)
	fs.Usage = func() { printUsage(fs.Output()) }
	registerFlags(fs, o)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return o, nil
}

func tiersFor(o *options, extraCount int) (tier1, tier2, tier3, doRecycle bool) {
	hasSpecific := o.dev || o.admin || o.recycle || extraCount > 0
	return (!hasSpecific || o.all), (o.dev || o.all), (o.admin || o.all), o.recycle
}

func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseFlags(args)
	if err != nil {
		return 2
	}

	if opts.version {
		fmt.Fprintf(stdout, "clean-my-disk v%s\n", model.Version)
		return 0
	}

	if opts.admin && !opts.dryRun && !opts.scan && !opts.bigFiles && !service.IsElevated() {
		if !elevate(stdout, stderr, opts.lang, args) {
			return 1
		}
		return 0
	}

	if len(args) == 0 && !opts.cli {
		p := tea.NewProgram(view.NewModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(stderr, "Error menjalankan TUI: %v\n", err)
			return 1
		}
		return 0
	}

	if opts.installSched {
		return schedule(stdout, stderr, opts.lang, false)
	}
	if opts.uninstallSched {
		return schedule(stdout, stderr, opts.lang, true)
	}

	home := config.Home()
	if home == "" {
		fmt.Fprintln(stderr, i18n.T(opts.lang, "fatal_home"))
		return 2
	}
	cfg := config.Load(home)

	svc := service.New(opts.dryRun)
	svc.HomeDir = home
	svc.Lang = opts.lang
	svc.Out = stdout
	svc.ThresholdDays = opts.days
	svc.MinSize = opts.minSizeMB << 20
	svc.Trash = opts.trash
	svc.KeepPatterns = cfg.KeepPatterns

	paths := append([]string(nil), opts.extra...)
	paths = append(paths, cfg.ExtraPaths...)

	if opts.scan {
		items := service.ScanAll(home, paths)
		if opts.json {
			free := service.Snapshot().DriveFree
			if err := view.EncodeJSON(stdout, model.ScanReport{
				Items:     items,
				Total:     model.TotalScanSize(items),
				DriveFree: free,
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			return 0
		}
		view.RenderScan(stdout, items, opts.lang)
		return 0
	}

	if opts.bigFiles {
		return runBigFiles(stdout, stderr, opts, home, cfg)
	}

	start := time.Now()
	var logs []model.LogEntry
	if opts.json {
		svc.OnLog = func(target, status string) { logs = append(logs, model.LogEntry{Target: target, Status: status}) }
	}

	tier1, tier2, tier3, doRecycle := tiersFor(opts, len(paths))
	if tier1 {
		svc.RunTier1()
	}
	if len(paths) > 0 {
		svc.RunExtras(paths)
	}
	if tier2 {
		svc.RunTier2()
	}
	if tier3 {
		svc.RunTier3()
	}
	if doRecycle {
		if !opts.json {
			fmt.Fprintln(stdout, "----------------------------------------------")
		}
		svc.RecycleBin()
	}

	elapsed := time.Since(start).Seconds()
	freed := svc.TotalFreed.Load()
	failures := svc.Failures.Load()

	if opts.json {
		if err := view.EncodeJSON(stdout, model.RunReport{
			ElapsedSeconds: elapsed,
			FreedBytes:     freed,
			Failures:       failures,
			DryRun:         opts.dryRun,
			Logs:           logs,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		view.RenderDone(stdout, elapsed, freed, failures, opts.dryRun, opts.lang)
	}

	notifyIfRequested(stderr, opts, freed, failures)

	if failures > 0 {
		return 1
	}
	return 0
}

func runBigFiles(stdout, stderr io.Writer, opts *options, home string, cfg *config.Config) int {
	svc := service.New(opts.dryRun)
	svc.HomeDir = home
	svc.Lang = opts.lang
	svc.Out = stdout
	svc.ThresholdDays = opts.days
	svc.MinSize = opts.minSizeMB << 20
	svc.Trash = opts.trash
	svc.KeepPatterns = cfg.KeepPatterns

	files := svc.FindBigFiles(home)

	if opts.json {
		if err := view.EncodeJSON(stdout, model.BigFileReport{
			Items: files,
			Total: model.TotalBigFileSize(files),
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	view.RenderBigFiles(stdout, files, opts.lang)
	if len(files) == 0 {
		return 0
	}
	fmt.Fprintf(stdout, i18n.T(opts.lang, "big_found")+"\n",
		len(files), model.HumanSize(model.TotalBigFileSize(files)))
	if opts.dryRun {
		return 0
	}

	var selected []model.BigFile
	if stdinIsTerminal() {
		fmt.Fprintln(stdout, i18n.T(opts.lang, "big_prompt"))
		fmt.Fprint(stdout, "> ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		idx := parseBigSelection(line, len(files))
		if idx == nil {
			return 0
		}
		for _, i := range idx {
			selected = append(selected, files[i])
		}
	} else {
		fmt.Fprintln(stdout, i18n.T(opts.lang, "big_auto"))
		selected = files
	}
	if len(selected) == 0 {
		return 0
	}

	start := time.Now()
	svc.DeleteBigFiles(selected)
	elapsed := time.Since(start).Seconds()
	freed := svc.TotalFreed.Load()
	failures := svc.Failures.Load()

	view.RenderDone(stdout, elapsed, freed, failures, false, opts.lang)
	notifyIfRequested(stderr, opts, freed, failures)
	if failures > 0 {
		return 1
	}
	return 0
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// parseBigSelection mengubah jawaban user menjadi indeks 0-based terpilih
// (urut input, dedupe). Nil balik nil = batal (baris kosong, 'q', atau input
// tidak valid).
func parseBigSelection(line string, max int) []int {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(line)), func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	if len(fields) == 0 {
		return nil
	}
	switch fields[0] {
	case "a", "all", "y", "yes":
		out := make([]int, max)
		for i := range out {
			out[i] = i
		}
		return out
	case "q", "quit", "c", "cancel", "n", "no":
		return nil
	}
	seen := map[int]bool{}
	var out []int
	for _, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 1 || n > max {
			return nil
		}
		if seen[n-1] {
			continue
		}
		seen[n-1] = true
		out = append(out, n-1)
	}
	return out
}

func notifyIfRequested(stderr io.Writer, opts *options, freed, failures int64) {
	if !opts.notify || opts.dryRun || opts.json || opts.scan {
		return
	}
	// Toast unpackaged hanya ditampilkan Windows bila shortcut Start Menu
	// ber-AUMID cocok ada — daftarkan dulu (idempoten) bila executable diketahui.
	exe, err := os.Executable()
	if err == nil {
		if regErr := service.EnsureToastRegistration(exe); regErr != nil {
			fmt.Fprintf(stderr, i18n.T(opts.lang, "toast_reg_failed")+" %v\n", regErr)
		}
	} else {
		fmt.Fprintf(stderr, i18n.T(opts.lang, "toast_reg_failed")+" %v\n", err)
	}
	body := fmt.Sprintf(i18n.T(opts.lang, "toast_freed"), model.HumanSize(freed))
	if failures > 0 {
		body += "\n" + fmt.Sprintf(i18n.T(opts.lang, "toast_failures"), failures)
	}
	if err := service.ShowToast(i18n.T(opts.lang, "toast_title"), body); err != nil {
		fmt.Fprintf(stderr, i18n.T(opts.lang, "notify_failed")+" %v\n", err)
	}
}

func elevate(stdout, stderr io.Writer, lang string, args []string) bool {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, i18n.T(lang, "sched_find_err")+" %v\n", err)
		return false
	}
	fmt.Fprintln(stdout, i18n.T(lang, "elevate_msg"))
	if err := service.RelaunchElevated(exe, args); err != nil {
		fmt.Fprintln(stderr, i18n.T(lang, "elevate_cancel"))
		return false
	}
	return true
}

func schedule(stdout, stderr io.Writer, lang string, remove bool) int {
	if remove {
		if err := service.RemoveSchedule(); err != nil {
			fmt.Fprintln(stderr, i18n.T(lang, "sched_delete_err"), err)
			return 1
		}
		fmt.Fprintln(stdout, i18n.T(lang, "sched_removed"))
		return 0
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(stderr, i18n.T(lang, "sched_find_err"), err)
		return 1
	}
	if err := service.EnsureToastRegistration(exe); err != nil {
		fmt.Fprintln(stderr, i18n.T(lang, "toast_reg_failed"), err)
	}
	if err := service.InstallWeeklySchedule(exe); err != nil {
		fmt.Fprintln(stderr, i18n.T(lang, "sched_create_err"), err)
		return 1
	}
	fmt.Fprintln(stdout, i18n.T(lang, "sched_created"))
	return 0
}
