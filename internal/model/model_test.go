package model

import "testing"

func TestHumanSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := HumanSize(tt.bytes)
		if got != tt.expected {
			t.Errorf("HumanSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestDriveUsedPct(t *testing.T) {
	d := Drive{Letter: "C", Free: 250, Total: 1000}
	if pct := d.UsedPct(); pct != 75.0 {
		t.Errorf("expected 75%%, got %.1f%%", pct)
	}

	dEmpty := Drive{Letter: "D", Free: 0, Total: 0}
	if pct := dEmpty.UsedPct(); pct != 0.0 {
		t.Errorf("expected 0%% on zero total, got %.1f%%", pct)
	}
}

func TestAggregateDrives(t *testing.T) {
	drives := []Drive{
		{Letter: "C", Free: 200, Total: 1000},
		{Letter: "D", Free: 300, Total: 1000},
	}
	free, total, pct := AggregateDrives(drives)
	if free != 500 || total != 2000 || pct != 75.0 {
		t.Errorf("got free=%d total=%d pct=%.1f; want free=500 total=2000 pct=75.0", free, total, pct)
	}

	free, total, pct = AggregateDrives(nil)
	if free != 0 || total != 0 || pct != 0.0 {
		t.Errorf("got free=%d total=%d pct=%.1f; want 0", free, total, pct)
	}
}

func TestTotalScanSize(t *testing.T) {
	items := []ScanItem{
		{Path: "a", Size: 100},
		{Path: "b", Size: 250},
	}
	if total := TotalScanSize(items); total != 350 {
		t.Errorf("expected 350, got %d", total)
	}
	if total := TotalScanSize(nil); total != 0 {
		t.Errorf("expected 0 on nil, got %d", total)
	}
}
