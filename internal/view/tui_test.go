package view

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"clean-my-disk/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(r ...rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: r}
}

func TestNewModel(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	m := NewModel()
	if m.state != stateSelect {
		t.Errorf("state awal harus stateSelect, got %v", m.state)
	}
	if len(m.items) != 7 {
		t.Fatalf("harus ada 7 item, got %d", len(m.items))
	}
	last := m.items[len(m.items)-1]
	if !strings.Contains(last.title, "Big Files") {
		t.Errorf("item terakhir harus Big Files, got %q", last.title)
	}
	if last.selected {
		t.Error("Big Files tidak boleh terpilih secara default")
	}
	for i, it := range m.items {
		want := i < 3
		if it.selected != want {
			t.Errorf("items[%d].selected = %v, ingin %v", i, it.selected, want)
		}
	}
	if m.dryRun {
		t.Error("dryRun awal harus false")
	}
}

func TestToggleDryRun(t *testing.T) {
	m := NewModel()

	mm, _ := m.Update(keyRunes('d'))
	m = mm.(Model)
	if !m.dryRun {
		t.Error("tekan 'd' harus mengaktifkan dry-run")
	}

	mm, _ = m.Update(keyRunes('d'))
	m = mm.(Model)
	if m.dryRun {
		t.Error("tekan 'd' kedua kali harus menonaktifkan dry-run")
	}
}

func TestToggleItemByNumber(t *testing.T) {
	m := NewModel()
	if m.items[3].selected {
		t.Fatal("item 4 (Tier 2) harusnya tidak terpilih awal")
	}

	mm, _ := m.Update(keyRunes('4'))
	m = mm.(Model)
	if !m.items[3].selected {
		t.Error("tekan '4' harus memilih item ke-4")
	}

	mm, _ = m.Update(keyRunes('4'))
	m = mm.(Model)
	if m.items[3].selected {
		t.Error("tekan '4' dua kali harus membatalkan pilihan")
	}
}

func TestToggleItemBySpace(t *testing.T) {
	m := NewModel()
	m.cursor = 5
	mm, _ := m.Update(keyRunes(' '))
	m = mm.(Model)
	if !m.items[5].selected {
		t.Error("space harus toggle item di cursor")
	}
}

func TestSelectAllToggle(t *testing.T) {
	m := NewModel()

	mm, _ := m.Update(keyRunes('a'))
	m = mm.(Model)
	for i, it := range m.items {
		if !it.selected {
			t.Errorf("setelah 'a', items[%d] harus terpilih", i)
		}
	}

	mm, _ = m.Update(keyRunes('a'))
	m = mm.(Model)
	for i, it := range m.items {
		if it.selected {
			t.Errorf("setelah 'a' dua kali, items[%d] harus tidak terpilih", i)
		}
	}
}

func TestEnterStartsRunning(t *testing.T) {
	m := NewModel()
	for i := range m.items {
		m.items[i].selected = false
	}

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)

	if m.state != stateRunning {
		t.Errorf("enter harus pindah ke stateRunning, got %v", m.state)
	}
	if m.totalTasks != 1 {
		t.Errorf("totalTasks harus 1 (guard nol-pilihan), got %d", m.totalTasks)
	}

	_ = cmd
}

func TestEnterLiveAsksConfirm(t *testing.T) {
	m := NewModel()
	if m.dryRun {
		t.Fatal("default harus Live")
	}
	if !m.anySelected() {
		t.Fatal("harus ada item terpilih default")
	}

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateConfirm {
		t.Errorf("enter Live harus ke stateConfirm, got %v", m.state)
	}

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateRunning {
		t.Errorf("enter di konfirmasi harus menjalankan, got %v", m.state)
	}
	_ = cmd
}

func TestConfirmEscCancels(t *testing.T) {
	m := NewModel()
	before := m.items[0].selected

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateConfirm {
		t.Fatalf("harus stateConfirm, got %v", m.state)
	}

	mm, _ = m.Update(keyRunes('q'))
	m = mm.(Model)
	if m.state != stateSelect {
		t.Errorf("esc/q harus batal ke stateSelect, got %v", m.state)
	}
	if m.items[0].selected != before {
		t.Error("pilihan tidak boleh berubah saat batal")
	}
}

func TestDryRunEnterRunsDirectly(t *testing.T) {
	m := NewModel()
	m.dryRun = true

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateRunning {
		t.Errorf("enter dry-run harus langsung running, got %v", m.state)
	}
	_ = cmd
}

func TestConfirmViewRenders(t *testing.T) {
	m := NewModel()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	v := m.View()
	for _, want := range []string{"Konfirmasi pembersihan", "Mode: Live", "Enter = jalankan"} {
		if !strings.Contains(v, want) {
			t.Errorf("View confirm harus memuat %q", want)
		}
	}
}

func TestSelectedEstimateShown(t *testing.T) {
	m := NewModel()
	sz := make([]int64, len(m.items))
	sz[0] = 2 << 20
	sz[1] = 1 << 20
	m.estSizes = sz

	v := m.View()
	if !strings.Contains(v, "Terpilih: 3 item") {
		t.Errorf("View harus menampilkan jumlah terpilih, got:\n%s", v)
	}
	if !strings.Contains(v, "3.0 MB") {
		t.Errorf("agregat estimasi 3.0 MB harus tampil, got:\n%s", v)
	}
}

func TestCatMsgUpdatesBreakdown(t *testing.T) {
	m := NewModel()
	mm, _ := m.Update(catMsg{cat: 0, freed: 2 << 20, fails: 1})
	m = mm.(Model)
	if m.freedByCat[0] != 2<<20 || m.failByCat[0] != 1 {
		t.Errorf("catMsg tidak terakumulasi: %+v %+v", m.freedByCat, m.failByCat)
	}
	if m.freedSoFar() != 2<<20 || m.failsSoFar() != 1 {
		t.Errorf("total freed/fails salah: %d %d", m.freedSoFar(), m.failsSoFar())
	}
}

func TestDoneViewShowsBreakdown(t *testing.T) {
	m := NewModel()
	mm, _ := m.Update(catMsg{cat: 0, freed: 2 << 20, fails: 1})
	m = mm.(Model)
	mm, _ = m.Update(doneMsg{elapsed: time.Second, freed: 2 << 20})
	m = mm.(Model)

	v := m.View()
	for _, want := range []string{"Rincian per kategori", "2.0 MB", "1 gagal"} {
		if !strings.Contains(v, want) {
			t.Errorf("View done harus memuat %q", want)
		}
	}
}

func TestRunningViewShowsLiveFreed(t *testing.T) {
	m := NewModel()
	m.state = stateRunning
	mm, _ := m.Update(catMsg{cat: 1, freed: 512})
	m = mm.(Model)

	v := m.View()
	if !strings.Contains(v, "Terbebas ~512 B") {
		t.Errorf("View running harus menampilkan freed live, got:\n%s", v)
	}
}

func TestDoneMsgSetsFinalState(t *testing.T) {
	m := NewModel()
	mm, _ := m.Update(doneMsg{elapsed: 2 * time.Second, freed: 12345})
	m = mm.(Model)

	if m.state != stateDone {
		t.Errorf("doneMsg harus pindah ke stateDone, got %v", m.state)
	}
	if m.totalFreed != 12345 {
		t.Errorf("totalFreed = %d, ingin 12345", m.totalFreed)
	}
	if m.elapsed != 2*time.Second {
		t.Errorf("elapsed = %v, ingin 2s", m.elapsed)
	}
}

func TestLogMsgAdvancesProgress(t *testing.T) {
	m := NewModel()
	m.state = stateRunning
	m.totalTasks = 5
	m.completedTasks = 0

	mm, _ := m.Update(logMsg{target: "C:\\cache", status: "OK"})
	m = mm.(Model)

	if m.completedTasks != 1 {
		t.Errorf("completedTasks = %d, ingin 1", m.completedTasks)
	}
	if m.currentAction != "C:\\cache" {
		t.Errorf("currentAction = %q, ingin path cache", m.currentAction)
	}
	if len(m.logs) != 1 {
		t.Errorf("harus ada 1 log, got %d", len(m.logs))
	}
}

func TestLogLevel(t *testing.T) {
	cases := []struct {
		status string
		want   int
	}{
		{"OK (~100 B)", 0},
		{"OK", 0},
		{"OK (~5 B, 2 terkunci dilewati)", 1},
		{"OK (~0 B, 1 folder kosong gagal)", 1},
		{"(lewati — butuh admin)", 2},
		{"(tidak ada)", 2},
		{"(akses ditolak / tidak terbaca)", 1},
		{"[hapus isi ~1.0 MB]", 2},
		{"WARNING: wuauserv failed to restart!", 1},
	}
	for _, c := range cases {
		if got := logLevel(c.status); got != c.want {
			t.Errorf("logLevel(%q) = %d, ingin %d", c.status, got, c.want)
		}
	}
}

func TestViewRendersSelection(t *testing.T) {
	m := NewModel()
	v := m.View()

	for _, want := range []string{"clean-my-disk", "Pilih target", "Mode: Live", "Tier 2: Dev Cache", "Recycle Bin", "Tekan 'd' untuk ganti mode"} {
		if !strings.Contains(v, want) {
			t.Errorf("View harus memuat %q", want)
		}
	}
}

func TestViewRendersDryRunBadge(t *testing.T) {
	m := NewModel()
	m.dryRun = true
	v := m.View()
	if !strings.Contains(v, "Simulasi (Dry-Run)") {
		t.Error("View dry-run harus menampilkan badge Simulasi (Dry-Run)")
	}
}

func TestViewRendersDone(t *testing.T) {
	m := NewModel()
	mm, _ := m.Update(doneMsg{elapsed: time.Second, freed: 999})
	m = mm.(Model)
	v := m.View()
	for _, want := range []string{"Pembersihan selesai", "Ruang hemat"} {
		if !strings.Contains(v, want) {
			t.Errorf("View done harus memuat %q", want)
		}
	}
}

func TestMouseClickItemToggle(t *testing.T) {
	m := NewModel()
	wasSelected := m.items[0].selected

	clickMsg := tea.MouseMsg{
		X:      10,
		Y:      5,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	mm, _ := m.Update(clickMsg)
	m = mm.(Model)

	if m.items[0].selected == wasSelected {
		t.Errorf("klik mouse pada item 0 harus mengubah selected dari %v", wasSelected)
	}
	if m.cursor != 0 {
		t.Errorf("klik mouse harus mengarahkan cursor ke 0, got %d", m.cursor)
	}
}

func TestMouseClickItemExpand(t *testing.T) {
	m := NewModel()
	clickMsg := tea.MouseMsg{
		X:      72,
		Y:      5,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	mm, _ := m.Update(clickMsg)
	m = mm.(Model)

	if m.expanded != 0 {
		t.Errorf("klik [Detail] pada item 0 harus membuka expanded = 0, got %d", m.expanded)
	}

	mm, _ = m.Update(clickMsg)
	m = mm.(Model)
	if m.expanded != -1 {
		t.Errorf("klik [Tutup] kedua kali harus menutup expanded = -1, got %d", m.expanded)
	}
}

func TestMouseClickBadges(t *testing.T) {
	m := NewModel()

	badgeRow := 5 + len(m.items) + 2
	clickDry := tea.MouseMsg{
		X:      10,
		Y:      badgeRow,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	mm, _ := m.Update(clickDry)
	m = mm.(Model)
	if !m.dryRun {
		t.Error("klik badge dry-run harus mengaktifkan dryRun")
	}

	clickTrash := tea.MouseMsg{
		X:      35,
		Y:      badgeRow,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	mm, _ = m.Update(clickTrash)
	m = mm.(Model)
	if !m.trash {
		t.Error("klik badge trash harus mengaktifkan trash")
	}

	clickLang := tea.MouseMsg{
		X:      60,
		Y:      badgeRow,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	mm, _ = m.Update(clickLang)
	m = mm.(Model)
	if m.lang != "en" {
		t.Errorf("klik badge lang harus mengganti ke en, got %s", m.lang)
	}
}

func TestMouseWheelNavigation(t *testing.T) {
	m := NewModel()
	m.cursor = 0

	wheelDown := tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
	}
	mm, _ := m.Update(wheelDown)
	m = mm.(Model)
	if m.cursor != 1 {
		t.Errorf("wheel down harus memajukan cursor ke 1, got %d", m.cursor)
	}

	wheelUp := tea.MouseMsg{
		Button: tea.MouseButtonWheelUp,
	}
	mm, _ = m.Update(wheelUp)
	m = mm.(Model)
	if m.cursor != 0 {
		t.Errorf("wheel up harus memundurkan cursor ke 0, got %d", m.cursor)
	}
}

func TestMouseClickDoneQuit(t *testing.T) {
	m := NewModel()
	m.state = stateDone

	clickAny := tea.MouseMsg{
		X:      15,
		Y:      10,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	_, cmd := m.Update(clickAny)
	if cmd == nil {
		t.Error("klik saat stateDone harus mengembalikan tea.Quit cmd")
	}
}

func makeBigModel(t *testing.T, n int) Model {
	t.Helper()
	m := NewModel()
	bigIdx := m.bigIdx
	if bigIdx < 0 || bigIdx >= len(m.items) {
		t.Fatalf("bigIdx tidak valid: %d", bigIdx)
	}
	files := make([]model.BigFile, 0, n)
	sel := make([]bool, 0, n)
	for i := 0; i < n; i++ {
		files = append(files, model.BigFile{
			Path:     fmt.Sprintf("C:\\dl\\setup-%d.exe", i),
			Size:     int64(i+1) << 20,
			Modified: time.Now().AddDate(0, 0, -40),
			Kind:     "exe",
		})
		sel = append(sel, false)
	}
	m.bigFiles = files
	m.bigSel = sel
	m.bigScanned = true
	m.estSizes = make([]int64, len(m.items))
	m.items[bigIdx].selected = true
	return m
}

func TestEnterBigFilesOpensPicker(t *testing.T) {
	m := makeBigModel(t, 3)
	for i := range m.items {
		if i != m.bigIdx {
			m.items[i].selected = false
		}
	}

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateBigPick {
		t.Fatalf("enter dengan Big Files terpilih harus ke stateBigPick, got %v", m.state)
	}
	if len(m.bigSel) != 3 {
		t.Errorf("bigSel harus 3, got %d", len(m.bigSel))
	}
}

func TestBigPickerNoCandidatesSkips(t *testing.T) {
	m := NewModel()
	m.bigScanned = true
	m.bigFiles = nil
	m.bigSel = nil
	m.items[m.bigIdx].selected = true
	for i := range m.items {
		if i != m.bigIdx {
			m.items[i].selected = false
		}
	}

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateConfirm {
		t.Errorf("tanpa kandidat harus langsung stateConfirm (live), got %v", m.state)
	}
}

func TestBigPickerToggleKeys(t *testing.T) {
	m := makeBigModel(t, 3)
	m.state = stateBigPick

	mm, _ := m.Update(keyRunes(' '))
	m = mm.(Model)
	if !m.bigSel[0] {
		t.Error("space harus mencentang kandidat di cursor (0)")
	}

	mm, _ = m.Update(keyRunes(' '))
	m = mm.(Model)
	if m.bigSel[0] {
		t.Error("space kedua harus membatalkan centang")
	}

	mm, _ = m.Update(keyRunes('a'))
	m = mm.(Model)
	for i, on := range m.bigSel {
		if !on {
			t.Errorf("'a' harus mencentang semua, bigSel[%d] = %v", i, on)
		}
	}

	mm, _ = m.Update(keyRunes('n'))
	m = mm.(Model)
	for i, on := range m.bigSel {
		if on {
			t.Errorf("'n' harus mengosongkan semua, bigSel[%d] = %v", i, on)
		}
	}
}

func TestBigPickerArrowMovesCursor(t *testing.T) {
	m := makeBigModel(t, 3)
	m.state = stateBigPick

	mm, _ := m.Update(keyRunes('j'))
	m = mm.(Model)
	if m.bigCursor != 1 {
		t.Errorf("j harus memajukan bigCursor ke 1, got %d", m.bigCursor)
	}
	mm, _ = m.Update(keyRunes('k'))
	m = mm.(Model)
	if m.bigCursor != 0 {
		t.Errorf("k harus memundurkan bigCursor ke 0, got %d", m.bigCursor)
	}
}

func TestBigPickerEnterLiveConfirm(t *testing.T) {
	m := makeBigModel(t, 3)
	for i := range m.items {
		if i != m.bigIdx {
			m.items[i].selected = false
		}
	}
	m.state = stateBigPick
	m.bigSel = []bool{true, false, true}

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateConfirm {
		t.Fatalf("enter picker (live) harus ke stateConfirm, got %v", m.state)
	}
	if len(m.bigPick) != 2 {
		t.Errorf("bigPick harus 2 file terpilih, got %d", len(m.bigPick))
	}
	if m.estSizes[m.bigIdx] != (1<<20)+(3<<20) {
		t.Errorf("estimasi bigIdx harus ukuran terpilih (file 0+2), got %d", m.estSizes[m.bigIdx])
	}
}

func TestBigPickerEnterLiveNothingPicked(t *testing.T) {
	m := makeBigModel(t, 3)
	for i := range m.items {
		if i != m.bigIdx {
			m.items[i].selected = false
		}
	}
	m.state = stateBigPick

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateBigPick {
		t.Errorf("tanpa file terpilih & tanpa item lain harus tetap di picker, got %v", m.state)
	}
	if m.bigPick != nil {
		t.Errorf("bigPick harus nil (belum konfirmasi), got %v", m.bigPick)
	}
}

func TestBigPickerEnterDryRunRunsDirectly(t *testing.T) {
	m := makeBigModel(t, 3)
	m.dryRun = true
	m.state = stateBigPick
	m.bigSel = []bool{false, true, false}

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.state != stateRunning {
		t.Fatalf("enter picker (dry-run) harus langsung stateRunning, got %v", m.state)
	}
	if len(m.bigPick) != 1 || m.bigPick[0].Path != m.bigFiles[1].Path {
		t.Errorf("bigPick harus berisi file ke-2 saja, got %v", m.bigPick)
	}
	_ = cmd
}

func TestBigPickerEscReturnsToSelect(t *testing.T) {
	m := makeBigModel(t, 3)
	m.state = stateBigPick

	mm, _ := m.Update(keyRunes('q'))
	m = mm.(Model)
	if m.state != stateSelect {
		t.Errorf("q dari picker harus kembali ke stateSelect, got %v", m.state)
	}
}

func TestBigPickerViewRenders(t *testing.T) {
	m := makeBigModel(t, 3)
	m.state = stateBigPick
	m.bigSel = []bool{true, false, true}
	v := m.View()
	for _, want := range []string{"Pilih file besar untuk dihapus", "2/3", "setup-0.exe", "LIVE"} {
		if !strings.Contains(v, want) {
			t.Errorf("View picker harus memuat %q, got:\n%s", want, v)
		}
	}
}

func TestMouseRunWithBigFilesOpensPicker(t *testing.T) {
	m := makeBigModel(t, 2)
	for i := range m.items {
		if i != m.bigIdx {
			m.items[i].selected = false
		}
	}
	bottomRow := 5 + len(m.items) + 2
	clickRun := tea.MouseMsg{
		X:      50,
		Y:      bottomRow + 3,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	mm, _ := m.Update(clickRun)
	m = mm.(Model)
	if m.state != stateBigPick {
		t.Errorf("klik RUN dengan Big Files harus ke picker dulu, got %v", m.state)
	}
}

func TestWindowResizeResponsive(t *testing.T) {
	m := NewModel()

	mm, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = mm.(Model)
	if m.width != 60 {
		t.Errorf("width harus 60, got %d", m.width)
	}
	vNarrow := m.View()
	if !strings.Contains(vNarrow, "clean-my-disk") {
		t.Error("View pada 60 col harus memuat clean-my-disk")
	}

	mm, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = mm.(Model)
	if m.width != 100 {
		t.Errorf("width harus 100, got %d", m.width)
	}
	vWide := m.View()
	if !strings.Contains(vWide, "clean-my-disk") {
		t.Error("View pada 100 col harus memuat clean-my-disk")
	}
	for _, it := range m.items {
		shortTitle := it.title
		if len(shortTitle) > 20 {
			shortTitle = shortTitle[:20]
		}
		if !strings.Contains(vWide, shortTitle) {
			t.Errorf("View harus memuat judul '%s'", shortTitle)
		}
	}
}
