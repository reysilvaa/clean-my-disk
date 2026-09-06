package controller

import "testing"

func TestParseFlagsDefaults(t *testing.T) {
	o, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("parseFlags(nil) error: %v", err)
	}
	if o.version || o.dryRun || o.dev || o.admin || o.recycle || o.all || o.cli || o.scan || o.json || o.trash ||
		o.installSched || o.uninstallSched {
		t.Errorf("semua flag harus default false, got %+v", o)
	}
	if len(o.extra) != 0 || o.minSizeMB != 0 || o.days != 0 || o.lang != "id" {
		t.Errorf("nilai default salah: %+v", o)
	}
}

func TestParseFlagsPathRepeatable(t *testing.T) {
	o, err := parseFlags([]string{"--path", "C:\\cache1", "--path", "old:D:\\lama"})
	if err != nil {
		t.Fatalf("parseFlags error: %v", err)
	}
	if len(o.extra) != 2 || o.extra[0] != "C:\\cache1" || o.extra[1] != "old:D:\\lama" {
		t.Errorf("isi extra salah: %v", o.extra)
	}
}

func TestParseFlagsSets(t *testing.T) {
	o, err := parseFlags([]string{"--dry-run", "--all", "--recycle", "--cli", "--scan", "--json", "--trash",
		"--min-size", "50", "--days", "30", "--lang", "en"})
	if err != nil {
		t.Fatalf("parseFlags error: %v", err)
	}
	if !o.dryRun || !o.all || !o.recycle || !o.cli || !o.scan || !o.json || !o.trash {
		t.Errorf("flag bool tidak terset: %+v", o)
	}
	if o.minSizeMB != 50 || o.days != 30 || o.lang != "en" {
		t.Errorf("flag numerik/string salah: %+v", o)
	}
	if o.dev || o.admin {
		t.Errorf("flag dev/admin tidak boleh terset: %+v", o)
	}
}

func TestParseFlagsRejectsUnknown(t *testing.T) {
	if _, err := parseFlags([]string{"--bogus"}); err == nil {
		t.Error("flag tak dikenal harus menghasilkan error")
	}
}

func TestParseFlagsMissingPathValue(t *testing.T) {
	if _, err := parseFlags([]string{"--path"}); err == nil {
		t.Error("--path tanpa nilai harus menghasilkan error")
	}
}

func TestParseFlagsVersion(t *testing.T) {
	o, err := parseFlags([]string{"--version"})
	if err != nil {
		t.Fatalf("parseFlags --version error: %v", err)
	}
	if !o.version {
		t.Error("o.version harus true")
	}
}

func TestTiersFor(t *testing.T) {
	cases := []struct {
		name  string
		o     *options
		extra int
		t1    bool
		t2    bool
		t3    bool
		rec   bool
	}{
		{"default (Tier 1 saja)", &options{}, 0, true, false, false, false},
		{"--dev saja", &options{dev: true}, 0, false, true, false, false},
		{"--admin saja", &options{admin: true}, 0, false, false, true, false},
		{"--recycle saja", &options{recycle: true}, 0, false, false, false, true},
		{"--all", &options{all: true}, 0, true, true, true, false},
		{"--all --recycle", &options{all: true, recycle: true}, 0, true, true, true, true},
		{"--dev --admin", &options{dev: true, admin: true}, 0, false, true, true, false},
		{"--path saja (non-hardcode)", &options{}, 1, false, false, false, false},
		{"--dev + --path", &options{dev: true}, 1, false, true, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t1, t2, t3, rec := tiersFor(c.o, c.extra)
			if t1 != c.t1 || t2 != c.t2 || t3 != c.t3 || rec != c.rec {
				t.Errorf("tiersFor = (%v,%v,%v,%v), ingin (%v,%v,%v,%v)",
					t1, t2, t3, rec, c.t1, c.t2, c.t3, c.rec)
			}
		})
	}
}
