package view

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"clean-my-disk/internal/i18n"
	"clean-my-disk/internal/model"
	"clean-my-disk/internal/service"
)

func renderDrives(drives []model.Drive, verbose bool, lang string) string {
	parts := make([]string, 0, len(drives))
	for _, d := range drives {
		if verbose {
			parts = append(parts, fmt.Sprintf(i18n.T(lang, "drive_info"),
				d.Letter, model.HumanSize(d.Free), model.HumanSize(d.Total), d.UsedPct()))
		} else {
			parts = append(parts, fmt.Sprintf(i18n.T(lang, "drive_free"),
				d.Letter, model.HumanSize(d.Free)))
		}
	}
	return strings.Join(parts, " · ")
}

func RenderScan(w io.Writer, items []model.ScanItem, lang string) {
	sorted := append([]model.ScanItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Size > sorted[j].Size })

	total := model.TotalScanSize(sorted)

	fmt.Fprintln(w, i18n.T(lang, "scan_header"))
	for i, it := range sorted {
		if i >= 10 {
			break
		}
		fmt.Fprintf(w, "%-44s %12s\n", it.Path, model.HumanSize(it.Size))
	}
	fmt.Fprintf(w, i18n.T(lang, "scan_total")+"\n", model.HumanSize(total))
	if snap := service.Snapshot(); len(snap.Drives) > 0 {
		fmt.Fprintln(w, renderDrives(snap.Drives, true, lang))
	}
}

func RenderBigFiles(w io.Writer, files []model.BigFile, lang string) {
	fmt.Fprintln(w, i18n.T(lang, "big_hdr"))
	if len(files) == 0 {
		fmt.Fprintln(w, i18n.T(lang, "big_none"))
		return
	}
	fmt.Fprintf(w, "%-4s %11s  %-10s  %s\n",
		i18n.T(lang, "big_col_idx"), i18n.T(lang, "big_col_size"),
		i18n.T(lang, "big_col_date"), i18n.T(lang, "big_col_path"))
	for i, f := range files {
		fmt.Fprintf(w, "%-4d %11s  %-10s  %s\n",
			i+1, model.HumanSize(f.Size), f.Modified.Format("2006-01-02"), f.Path)
	}
}

func RenderDone(w io.Writer, elapsed float64, freed, failures int64, dryRun bool, lang string) {
	fmt.Fprintln(w, "----------------------------------------------")
	fmt.Fprintf(w, i18n.T(lang, "done_seconds")+"\n", elapsed)
	if !dryRun && freed > 0 {
		fmt.Fprintf(w, i18n.T(lang, "done_freed")+"\n", model.HumanSize(freed))
	}
	if failures > 0 {
		fmt.Fprintf(w, i18n.T(lang, "done_failures")+"\n", failures)
	}
}

func EncodeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
