package model

import "fmt"

var Version = "1.5.0"

const OldFileThresholdDays = 14

type ScanItem struct {
	Path string
	Size int64
	Kind string
}

type LogEntry struct {
	Target string
	Status string
}

type RunReport struct {
	ElapsedSeconds float64
	FreedBytes     int64
	Failures       int64
	DryRun         bool
	Logs           []LogEntry
}

func HumanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type Drive struct {
	Letter string
	Free   int64
	Total  int64
}

func (d Drive) UsedPct() float64 {
	if d.Total <= 0 {
		return 0
	}
	return float64(d.Total-d.Free) / float64(d.Total) * 100
}

func AggregateDrives(drives []Drive) (free, total int64, percentUsed float64) {
	for _, d := range drives {
		free += d.Free
		total += d.Total
	}
	if total == 0 {
		return 0, 0, 0
	}
	return free, total, float64(total-free) / float64(total) * 100
}

type SystemInfo struct {
	Drives     []Drive
	DriveFree  int64
	DriveTotal int64
	UsedPct    float64
	IsAdmin    bool
}

type ScanReport struct {
	Items     []ScanItem
	Total     int64
	DriveFree int64
}

func TotalScanSize(items []ScanItem) int64 {
	var total int64
	for _, it := range items {
		total += it.Size
	}
	return total
}
