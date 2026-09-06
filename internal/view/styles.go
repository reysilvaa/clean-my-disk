package view

import "github.com/charmbracelet/lipgloss"

var (
	claudeSparkle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))

	brandTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC"))

	brandVersion = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E"))

	subTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E"))

	sectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC"))

	divider = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#30363D"))

	cursorActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))

	checkActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#3FB950"))

	checkInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#484F58"))

	itemActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC"))

	itemInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C9D1D9"))

	itemDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E"))

	badgeLive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#3FB950")).
			Background(lipgloss.Color("#121D17")).
			Padding(0, 1)

	badgeDry = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D29922")).
			Background(lipgloss.Color("#211B0D")).
			Padding(0, 1)

	badgeAdmin = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A371F7")).
			Background(lipgloss.Color("#1F182E")).
			Padding(0, 1)

	cardBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Padding(0, 1)

	cardBoxSuccess = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#238636")).
			Padding(0, 1)

	statCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(0, 1).
			Align(lipgloss.Center)

	statCardTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E")).
			Bold(true)

	statCardValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F0F6FC")).
			Bold(true)

	driveCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(0, 1)

	btnNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C9D1D9")).
			Background(lipgloss.Color("#21262D")).
			Padding(0, 1)

	btnAccent = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#1F6FEB")).
			Padding(0, 1)

	btnSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#238636")).
			Padding(0, 1)

	statusLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E"))

	statusValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F0F6FC"))

	barFilled = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#58A6FF"))

	barEmpty = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#21262D"))

	barGreen = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3FB950"))

	percentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))

	logTarget = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C9D1D9"))

	logSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3FB950"))

	logDim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6E7681"))

	logFail = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F85149"))

	footerKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C9D1D9"))

	footerDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7681"))

	successCheck = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#3FB950"))

	failCheck = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F85149"))
)
