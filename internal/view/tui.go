package view

import (
	"fmt"
	"strings"
	"time"

	"clean-my-disk/internal/config"
	"clean-my-disk/internal/model"
	"clean-my-disk/internal/service"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateSelect state = iota
	stateRunning
	stateDone
)

type targetItem struct {
	title       string
	description string
	selected    bool
	count       int
	subPaths    []string
}

type logMsg struct {
	target string
	status string
}

type doneMsg struct {
	elapsed time.Duration
	freed   int64
}

type scanMsg struct {
	sizes []int64
}

type Model struct {
	width          int
	height         int
	state          state
	cursor         int
	items          []targetItem
	dryRun         bool
	trash          bool
	lang           string
	expanded       int
	logs           []string
	scrollOffset   int
	totalFreed     int64
	elapsed        time.Duration
	totalTasks     int
	completedTasks int
	currentAction  string
	homeDir        string
	drives         []model.Drive
	isAdmin        bool
	estSizes       []int64
	sub            chan tea.Msg
}

func NewModel() Model {
	snap := service.Snapshot()
	home := config.Home()
	cfg := config.Load(home)

	items := []targetItem{
		{
			title:       "Tier 1: Cache Regenerable",
			description: "Temp, GPU, Dev (npm/bun/go), IDE",
			selected:    true,
			count:       len(config.DefaultTier1Paths(home)),
			subPaths:    config.DefaultTier1Paths(home),
		},
		{
			title:       "Tier 1: Browser Cache",
			description: "Chrome, Edge, Brave, Firefox, Opera, dll",
			selected:    true,
			count:       len(config.BrowserUserDirs(home)),
			subPaths:    []string{"Chrome", "Edge", "Brave", "Firefox", "Opera", "Vivaldi", "Arc", "Zen"},
		},
		{
			title:       "Tier 1: Sampah Updater",
			description: "File updater lama > 14 hari",
			selected:    true,
			count:       len(config.DefaultUpdaterPaths(home)),
			subPaths:    config.DefaultUpdaterPaths(home),
		},
		{
			title:       "Tier 2: Dev Cache",
			description: "Gradle & Cargo (akan download ulang)",
			selected:    false,
			count:       2,
			subPaths:    []string{config.GradleCacheDir(home), config.CargoRegistryDir(home)},
		},
		{
			title:       "Tier 3: Sistem (Admin)",
			description: "Temp, WinUpdate, CBS logs, Dump",
			selected:    false,
			count:       len(config.DefaultTier3Paths()),
			subPaths:    config.DefaultTier3Paths(),
		},
		{
			title:       "Recycle Bin",
			description: "Kosongkan Recycle Bin semua drive",
			selected:    false,
			count:       1,
			subPaths:    []string{"$Recycle.Bin (Semua Drive)"},
		},
	}

	if len(cfg.ExtraPaths) > 0 {
		items = append(items, targetItem{
			title:       "Path Kustom",
			description: "~/.clean-caches.paths",
			selected:    false,
			count:       len(cfg.ExtraPaths),
			subPaths:    cfg.ExtraPaths,
		})
	}

	return Model{
		width:        92,
		height:       26,
		state:        stateSelect,
		dryRun:       false,
		trash:        false,
		lang:         "id",
		expanded:     -1,
		homeDir:      home,
		drives:       snap.Drives,
		isAdmin:      snap.IsAdmin,
		items:        items,
		sub:          make(chan tea.Msg, 100),
		scrollOffset: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return m.startScan()
}

func (m Model) startScan() tea.Cmd {
	return func() tea.Msg {
		return scanMsg{sizes: service.TierEstimates(m.homeDir)}
	}
}

func waitForMsg(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func (m Model) contentWidth() int {
	w := m.width
	if w <= 0 {
		w = 92
	}
	w -= 4
	if w < 56 {
		w = 56
	}
	if w > 110 {
		w = 110
	}
	return w
}

func logLevel(status string) int {
	s := strings.ToLower(status)
	if containsAny(s, "gagal", "terkunci", "peringatan", "warning", "failed", "locked", "error", "denied", "ditolak") {
		return 1
	}
	if strings.HasPrefix(strings.TrimSpace(s), "ok") {
		return 0
	}
	return 2
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func indent(s string, spaces int) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) formatLogLine(target, status string, innerW int) string {
	if innerW < 40 {
		innerW = 78
	}
	short := target
	if m.homeDir != "" && strings.HasPrefix(short, m.homeDir) {
		short = "~" + short[len(m.homeDir):]
	}

	available := innerW - 5
	targetW := available * 56 / 100
	statusW := available - targetW
	if targetW < 18 {
		targetW = 18
	}
	if statusW < 14 {
		statusW = 14
	}

	if len(short) > targetW {
		half := (targetW - 3) / 2
		short = short[:half] + "..." + short[len(short)-(targetW-3-half):]
	}

	stat := status
	if len(stat) > statusW {
		stat = stat[:statusW-3] + "..."
	}

	colTarget := fmt.Sprintf("%-*s", targetW, short)
	colStatus := fmt.Sprintf("%-*s", statusW, stat)

	switch logLevel(status) {
	case 1:
		return fmt.Sprintf(" %s %s %s", failCheck.Render("✗"), logTarget.Render(colTarget), logFail.Render(colStatus))
	case 2:
		return fmt.Sprintf(" %s %s %s", logDim.Render("·"), logDim.Render(colTarget), logDim.Render(colStatus))
	default:
		return fmt.Sprintf(" %s %s %s", successCheck.Render("✓"), logTarget.Render(colTarget), logSuccess.Render(colStatus))
	}
}

func (m Model) renderProgressBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	filledLen := int(percent * float64(width))
	if filledLen > width {
		filledLen = width
	}
	emptyLen := width - filledLen

	bar := barFilled.Render(strings.Repeat("█", filledLen)) + barEmpty.Render(strings.Repeat("░", emptyLen))
	pctText := percentStyle.Render(fmt.Sprintf("%3.0f%%", percent*100))
	countText := itemDesc.Render(fmt.Sprintf("(%d/%d item)", m.completedTasks, m.totalTasks))

	return fmt.Sprintf("[%s] %s  %s", bar, pctText, countText)
}

func (m Model) triggerRun() (tea.Model, tea.Cmd) {
	m.totalTasks = 0
	m.completedTasks = 0
	for _, it := range m.items {
		if it.selected {
			m.totalTasks += it.count
		}
	}
	if m.totalTasks == 0 {
		m.totalTasks = 1
	}

	m.state = stateRunning
	return m, tea.Batch(
		m.startCleaning(),
		waitForMsg(m.sub),
	)
}

func (m Model) inspectorLines(idx int) []string {
	if idx < 0 || idx >= len(m.items) {
		idx = 0
	}
	item := m.items[idx]

	var lines []string
	lines = append(lines, sectionHeader.Render("Detail & Keamanan Target"))
	lines = append(lines, divider.Render("─────────────────────────────────"))

	plainTitle := item.title
	if len(plainTitle) > 28 {
		plainTitle = plainTitle[:25] + "..."
	}
	lines = append(lines, statusLabel.Render("Target:    ")+statusValue.Render(plainTitle))

	estStr := "0 B"
	if m.estSizes != nil && idx < len(m.estSizes) && m.estSizes[idx] > 0 {
		estStr = model.HumanSize(m.estSizes[idx])
	}
	lines = append(lines, statusLabel.Render("Estimasi:  ")+percentStyle.Render("~"+estStr))

	riskLabel := statusLabel.Render("Kebijakan: ")
	switch idx {
	case 0, 1, 2:
		lines = append(lines, riskLabel+badgeLive.Render("Tier 1: Aman (Auto)"))
	case 3:
		lines = append(lines, riskLabel+badgeDry.Render("Tier 2: Unduh Ulang"))
	case 4:
		lines = append(lines, riskLabel+badgeAdmin.Render("Tier 3: Hak Admin"))
	case 5:
		lines = append(lines, riskLabel+failCheck.Render("Recycle Bin (Purge)"))
	default:
		lines = append(lines, riskLabel+itemDesc.Render("Path Kustom"))
	}

	lines = append(lines, statusLabel.Render("Direktori:")+itemDesc.Render(fmt.Sprintf(" (%d lokasi)", len(item.subPaths))))

	count := 0
	for _, p := range item.subPaths {
		if count >= 3 {
			lines = append(lines, itemDesc.Render(fmt.Sprintf("  ... dan %d lainnya", len(item.subPaths)-3)))
			break
		}
		pShort := p
		if m.homeDir != "" && strings.HasPrefix(pShort, m.homeDir) {
			pShort = "~" + pShort[len(m.homeDir):]
		}
		if len(pShort) > 28 {
			pShort = pShort[:13] + "..." + pShort[len(pShort)-12:]
		}
		lines = append(lines, logDim.Render("├── ")+itemDesc.Render(pShort))
		count++
	}

	return lines
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.state == stateSelect && m.cursor > 0 {
				m.cursor--
			} else if m.state == stateRunning || m.state == stateDone {
				if m.scrollOffset < len(m.logs)-5 {
					m.scrollOffset++
				}
			}
		case "down", "j":
			if m.state == stateSelect && m.cursor < len(m.items)-1 {
				m.cursor++
			} else if m.state == stateRunning || m.state == stateDone {
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			}
		case " ":
			if m.state == stateSelect {
				m.items[m.cursor].selected = !m.items[m.cursor].selected
			}
		case "a", "A":
			if m.state == stateSelect {
				allSelected := true
				for _, it := range m.items {
					if !it.selected {
						allSelected = false
						break
					}
				}
				for i := range m.items {
					m.items[i].selected = !allSelected
				}
			}
		case "1", "2", "3", "4", "5", "6", "7":
			if m.state == stateSelect {
				idx := int(msg.String()[0] - '1')
				if idx >= 0 && idx < len(m.items) {
					m.items[idx].selected = !m.items[idx].selected
				}
			}
		case "d", "D":
			if m.state == stateSelect {
				m.dryRun = !m.dryRun
			}
		case "t", "T":
			if m.state == stateSelect {
				m.trash = !m.trash
			}
		case "l", "L":
			if m.lang == "id" {
				m.lang = "en"
			} else {
				m.lang = "id"
			}
		case "e", "E", "tab":
			if m.state == stateSelect {
				if m.expanded == m.cursor {
					m.expanded = -1
				} else {
					m.expanded = m.cursor
				}
			}
		case "enter":
			switch m.state {
			case stateSelect:
				return m.triggerRun()
			case stateDone:
				return m, tea.Quit
			}
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			if m.state == stateSelect {
				if m.cursor > 0 {
					m.cursor--
				}
			} else {
				if m.scrollOffset < len(m.logs)-5 {
					m.scrollOffset++
				}
			}
			return m, nil
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if m.state == stateSelect {
				if m.cursor < len(m.items)-1 {
					m.cursor++
				}
			} else {
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			}
			return m, nil
		}

		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if m.state == stateSelect {
				const itemStartRow = 5
				row := msg.Y - itemStartRow
				curRow := 0
				for i := range m.items {
					itemHeight := 1
					if m.expanded == i {
						itemHeight += len(m.items[i].subPaths)
					}
					if row >= curRow && row < curRow+itemHeight {
						m.cursor = i
						if msg.X >= 42 || row > curRow {
							if m.expanded == i {
								m.expanded = -1
							} else {
								m.expanded = i
							}
						} else {
							m.items[i].selected = !m.items[i].selected
						}
						return m, nil
					}
					curRow += itemHeight
				}

				bottomRow := itemStartRow + curRow + 2
				if msg.Y >= bottomRow && msg.Y <= bottomRow+2 {
					if msg.X >= 2 && msg.X <= 28 {
						m.dryRun = !m.dryRun
						return m, nil
					}
					if msg.X >= 29 && msg.X <= 54 {
						m.trash = !m.trash
						return m, nil
					}
					if msg.X >= 55 && msg.X <= 75 {
						if m.lang == "id" {
							m.lang = "en"
						} else {
							m.lang = "id"
						}
						return m, nil
					}
				}
				if msg.Y >= bottomRow+3 {
					if msg.X >= 60 {
						return m, tea.Quit
					}
					if msg.X >= 44 && msg.X < 60 {
						return m.triggerRun()
					}
					if msg.X >= 34 && msg.X < 44 {
						if m.lang == "id" {
							m.lang = "en"
						} else {
							m.lang = "id"
						}
						return m, nil
					}
					if msg.X >= 24 && msg.X < 34 {
						m.trash = !m.trash
						return m, nil
					}
					if msg.X >= 14 && msg.X < 24 {
						m.dryRun = !m.dryRun
						return m, nil
					}
					if msg.X >= 6 && msg.X < 14 {
						if m.expanded == m.cursor {
							m.expanded = -1
						} else {
							m.expanded = m.cursor
						}
						return m, nil
					}
					if msg.X < 6 {
						m.items[m.cursor].selected = !m.items[m.cursor].selected
						return m, nil
					}
				}
			} else if m.state == stateDone {
				return m, tea.Quit
			}
			return m, nil
		}

	case scanMsg:
		m.estSizes = msg.sizes
		if m.state == stateRunning {
			return m, waitForMsg(m.sub)
		}

	case logMsg:
		innerW := m.contentWidth() - 4
		m.completedTasks++
		if m.completedTasks > m.totalTasks {
			m.completedTasks = m.totalTasks
		}
		m.currentAction = msg.target
		m.logs = append(m.logs, m.formatLogLine(msg.target, msg.status, innerW))
		if m.state == stateRunning {
			return m, waitForMsg(m.sub)
		}

	case doneMsg:
		m.completedTasks = m.totalTasks
		m.state = stateDone
		m.elapsed = msg.elapsed
		m.totalFreed = msg.freed
		return m, nil
	}

	return m, nil
}

func (m Model) startCleaning() tea.Cmd {
	return func() tea.Msg {
		go func() {
			start := time.Now()
			s := service.New(m.dryRun)
			s.Lang = m.lang
			s.Trash = m.trash
			s.OnLog = func(target, status string) {
				m.sub <- logMsg{target: target, status: status}
			}

			if len(m.items) > 0 && m.items[0].selected {
				s.RunTier1Regenerable()
			}
			if len(m.items) > 1 && m.items[1].selected {
				s.RunTier1Browsers()
			}
			if len(m.items) > 2 && m.items[2].selected {
				s.RunTier1Updaters()
			}
			if len(m.items) > 3 && m.items[3].selected {
				s.RunTier2()
			}
			if len(m.items) > 4 && m.items[4].selected {
				s.RunTier3()
			}
			if len(m.items) > 5 && m.items[5].selected {
				s.RecycleBin()
			}
			if len(m.items) > 6 && m.items[6].selected {
				cfg := config.Load(m.homeDir)
				s.RunExtras(cfg.ExtraPaths)
			}

			m.sub <- doneMsg{
				elapsed: time.Since(start),
				freed:   s.TotalFreed.Load(),
			}
		}()
		return nil
	}
}

func (m Model) View() string {
	var out strings.Builder
	w := m.contentWidth()
	divLine := divider.Render(strings.Repeat("─", w))

	out.WriteString("  ")
	out.WriteString(brandTitle.Render("clean-my-disk"))
	out.WriteString(" ")
	out.WriteString(brandVersion.Render("(v" + model.Version + ")"))
	out.WriteString(" · ")
	out.WriteString(subTitle.Render("Windows Cache & Junk Cleaner"))
	out.WriteByte('\n')

	if len(m.drives) > 0 {
		out.WriteString("    ")
		out.WriteString(itemDesc.Render(renderDrives(m.drives, true, m.lang)))
		out.WriteString("\n\n")
	} else {
		out.WriteString("\n")
	}

	switch m.state {
	case stateSelect:
		out.WriteString("  ")
		hdrText := "Pilih target pembersihan (klik mouse atau keyboard):"
		if m.lang == "en" {
			hdrText = "Select cleaning targets (click or use keyboard):"
		}
		out.WriteString(sectionHeader.Render(hdrText))
		out.WriteString("\n\n")

		var itemLines []string
		for i, item := range m.items {
			cur := "  "
			if m.cursor == i {
				cur = cursorActive.Render("❯ ")
			}

			chk := checkInactive.Render("[ ]")
			if item.selected {
				chk = checkActive.Render("[×]")
			}

			num := footerDesc.Render(fmt.Sprintf("%d.", i+1))

			plainTitle := item.title
			if len(plainTitle) > 26 {
				plainTitle = plainTitle[:23] + "..."
			}
			itemTitle := itemInactive.Render(fmt.Sprintf("%-26s", plainTitle))
			if m.cursor == i {
				itemTitle = itemActive.Render(fmt.Sprintf("%-26s", plainTitle))
			}

			est := ""
			if m.estSizes != nil && i < len(m.estSizes) && m.estSizes[i] > 0 {
				est = fmt.Sprintf("~%s", model.HumanSize(m.estSizes[i]))
			}
			colEst := statusValue.Render(fmt.Sprintf("%-8s", est))

			detailBtn := footerKey.Render("[Detail]")
			if m.expanded == i {
				detailBtn = cursorActive.Render("[Tutup]")
			}

			line := fmt.Sprintf("%s%s %s %s %s %s", cur, chk, num, itemTitle, colEst, detailBtn)
			itemLines = append(itemLines, line)

			if m.expanded == i {
				for _, sub := range item.subPaths {
					subShort := sub
					if m.homeDir != "" && strings.HasPrefix(subShort, m.homeDir) {
						subShort = "~" + subShort[len(m.homeDir):]
					}
					if len(subShort) > 36 {
						subShort = subShort[:16] + "..." + subShort[len(subShort)-17:]
					}
					itemLines = append(itemLines, fmt.Sprintf("      %s %s", logDim.Render("↳"), itemDesc.Render(subShort)))
				}
			}
		}

		if w >= 88 {
			leftW := 56
			rightW := w - leftW - 2
			targetPanel := cardBox.Copy().Width(leftW).Render(strings.Join(itemLines, "\n"))
			rightLines := m.inspectorLines(m.cursor)
			inspectorPanel := cardBox.Copy().Width(rightW).Render(strings.Join(rightLines, "\n"))
			sideBySide := lipgloss.JoinHorizontal(lipgloss.Top, targetPanel, "  ", inspectorPanel)
			out.WriteString(indent(sideBySide, 2))
			out.WriteString("\n")
		} else {
			targetPanel := cardBox.Copy().Width(w).Render(strings.Join(itemLines, "\n"))
			out.WriteString(indent(targetPanel, 2))
			out.WriteString("\n")
		}

		out.WriteString("\n  ")
		out.WriteString(divLine)
		out.WriteString("\n  ")

		modeStr := badgeLive.Render("Mode: Live")
		if m.dryRun {
			modeStr = badgeDry.Render("Mode: Simulasi (Dry-Run)")
		}
		trashStr := badgeLive.Render("Trash: Off")
		if m.trash {
			trashStr = badgeDry.Render("Trash: On (Recycle Bin)")
		}
		langStr := badgeLive.Render("Lang: " + strings.ToUpper(m.lang))

		adminStr := ""
		if m.isAdmin {
			adminStr = " · " + badgeAdmin.Render("ADMIN")
		}

		modeHint := "Tekan 'd' untuk ganti mode · 't' trash · 'l' bahasa"
		if m.lang == "en" {
			modeHint = "Press 'd' to change mode · 't' trash · 'l' language"
		}

		if w >= 78 {
			out.WriteString(fmt.Sprintf("%s · %s · %s%s · %s\n", modeStr, trashStr, langStr, adminStr, itemDesc.Render(modeHint)))
		} else {
			out.WriteString(fmt.Sprintf("%s · %s · %s%s\n  %s\n", modeStr, trashStr, langStr, adminStr, itemDesc.Render(modeHint)))
		}

		out.WriteString("  ")
		protNote := "Proteksi aktif: WSL, Docker, Database, data login browser tidak disentuh."
		if m.lang == "en" {
			protNote = "Active protection: WSL, Docker, Databases, browser login data untouched."
		}
		if len(protNote) > w {
			protNote = protNote[:w-3] + "..."
		}
		out.WriteString(itemDesc.Render(protNote))
		out.WriteString("\n\n  ")

		if w >= 78 {
			nav := fmt.Sprintf("%s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s",
				btnNormal.Render("↑/↓"), footerDesc.Render("nav"),
				btnNormal.Render("space"), footerDesc.Render("pilih"),
				btnNormal.Render("tab"), footerDesc.Render("detail"),
				btnNormal.Render("d"), footerDesc.Render("dry"),
				btnNormal.Render("t"), footerDesc.Render("trash"),
				btnNormal.Render("l"), footerDesc.Render("lang"),
				btnAccent.Render("enter"), footerDesc.Render("eksekusi"),
				btnNormal.Render("q"), footerDesc.Render("keluar"),
			)
			out.WriteString(nav)
		} else {
			nav1 := fmt.Sprintf("%s %s   %s %s   %s %s   %s %s",
				btnNormal.Render("↑/↓"), footerDesc.Render("nav"),
				btnNormal.Render("space"), footerDesc.Render("pilih"),
				btnNormal.Render("tab"), footerDesc.Render("detail"),
				btnNormal.Render("d"), footerDesc.Render("dry"),
			)
			nav2 := fmt.Sprintf("%s %s   %s %s   %s %s   %s %s",
				btnNormal.Render("t"), footerDesc.Render("trash"),
				btnNormal.Render("l"), footerDesc.Render("lang"),
				btnAccent.Render("enter"), footerDesc.Render("eksekusi"),
				btnNormal.Render("q"), footerDesc.Render("keluar"),
			)
			out.WriteString(nav1 + "\n  " + nav2)
		}
		out.WriteString("\n\n")

	case stateRunning:
		out.WriteString("  ")
		runHdr := "Sedang membersihkan cache..."
		if m.lang == "en" {
			runHdr = "Cleaning caches in progress..."
		}
		out.WriteString(sectionHeader.Render(runHdr))
		out.WriteString("\n\n  ")

		pct := 0.0
		if m.totalTasks > 0 {
			pct = float64(m.completedTasks) / float64(m.totalTasks)
		}
		barW := w - 36
		if barW < 16 {
			barW = 16
		}
		out.WriteString(m.renderProgressBar(pct, barW))
		out.WriteString("\n\n")

		displayLogs := m.logs
		if len(displayLogs) > 10 {
			end := len(displayLogs) - m.scrollOffset
			if end < 10 {
				end = 10
			}
			if end > len(displayLogs) {
				end = len(displayLogs)
			}
			start := end - 10
			if start < 0 {
				start = 0
			}
			displayLogs = displayLogs[start:end]
		}

		var logBlock strings.Builder
		for _, l := range displayLogs {
			logBlock.WriteString(l)
			logBlock.WriteByte('\n')
		}
		if logBlock.Len() > 0 {
			boxStyle := cardBox.Copy().Width(w)
			out.WriteString(indent(boxStyle.Render(strings.TrimRight(logBlock.String(), "\n")), 2))
			out.WriteString("\n")
		}

		out.WriteString("\n  ")
		out.WriteString(divLine)
		out.WriteString("\n  ")
		runSub := "Menjalankan operasi I/O paralel goroutines... (scroll/panah untuk log)"
		if m.lang == "en" {
			runSub = "Parallel I/O goroutines in progress... (scroll/arrows for logs)"
		}
		out.WriteString(footerDesc.Render(runSub))
		out.WriteString("\n\n")

	case stateDone:
		out.WriteString("  ")
		doneHdr := "Pembersihan selesai!"
		if m.lang == "en" {
			doneHdr = "Cleaning completed!"
		}
		out.WriteString(successCheck.Render("✓ ") + sectionHeader.Render(doneHdr))
		out.WriteString("\n\n")

		lblFreed := "Ruang hemat"
		lblTime := "Waktu proses"
		lblDone := "Item selesai"
		if m.lang == "en" {
			lblFreed = "Space saved"
			lblTime = "Time elapsed"
			lblDone = "Items done"
		}

		cardW := (w - 4) / 3
		if cardW < 18 {
			cardW = 18
		}
		statStyle := statCard.Copy().Width(cardW)
		card1 := statStyle.Render(fmt.Sprintf("%s\n\n%s", statCardTitle.Render(lblFreed), statCardValue.Render("~"+model.HumanSize(m.totalFreed))))
		card2 := statStyle.Render(fmt.Sprintf("%s\n\n%s", statCardTitle.Render(lblTime), statCardValue.Render(fmt.Sprintf("%.2fs", m.elapsed.Seconds()))))
		card3 := statStyle.Render(fmt.Sprintf("%s\n\n%s", statCardTitle.Render(lblDone), statCardValue.Render(fmt.Sprintf("%d / %d", m.completedTasks, m.totalTasks))))
		statCards := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2, "  ", card3)

		out.WriteString(indent(statCards, 2))
		out.WriteString("\n\n")

		displayLogs := m.logs
		if len(displayLogs) > 10 {
			end := len(displayLogs) - m.scrollOffset
			if end < 10 {
				end = 10
			}
			if end > len(displayLogs) {
				end = len(displayLogs)
			}
			start := end - 10
			if start < 0 {
				start = 0
			}
			displayLogs = displayLogs[start:end]
		}

		var logBlock strings.Builder
		for _, l := range displayLogs {
			logBlock.WriteString(l)
			logBlock.WriteByte('\n')
		}
		if logBlock.Len() > 0 {
			boxStyle := cardBox.Copy().Width(w)
			out.WriteString(indent(boxStyle.Render(strings.TrimRight(logBlock.String(), "\n")), 2))
			out.WriteString("\n")
		}

		out.WriteString("\n  ")
		out.WriteString(divLine)
		out.WriteString("\n\n  ")
		quitBtn := btnAccent.Render(" ↵ Selesai (Tekan Q atau klik di mana saja untuk keluar) ")
		out.WriteString(quitBtn)
		out.WriteString("\n\n")
	}

	return out.String()
}
