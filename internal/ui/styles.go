package ui

import "github.com/charmbracelet/lipgloss"

// Color palette — matches the shared design system.
var (
	colorAccent   = lipgloss.Color("#5F9FFF")
	colorText     = lipgloss.Color("#FFFFFF")
	colorBright   = lipgloss.Color("#FFFFFF")
	colorMuted    = lipgloss.Color("#A0AEC0")
	colorFaint    = lipgloss.Color("#718096")
	colorBorder   = lipgloss.Color("#4A5568")
	colorSelectBg = lipgloss.Color("#2C5282")
	colorOk       = lipgloss.Color("#68D391")
	colorWarn     = lipgloss.Color("#F6C950")
	colorErr      = lipgloss.Color("#FC8181")
)

// Layout styles.
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 2)

	separatorStyle = lipgloss.NewStyle().
			Foreground(colorBorder)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorFaint).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorWarn).
			Padding(0, 1)
)

// Tab styles.
var (
	tabActive = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 2)

	tabInactive = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2)
)

// Row styles.
var (
	rowNormalStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 2)

	rowSelectedStyle = lipgloss.NewStyle().
				Foreground(colorBright).
				Background(colorSelectBg).
				Bold(true).
				Padding(0, 2)

	rowDimStyle = lipgloss.NewStyle().
			Foreground(colorFaint).
			Padding(0, 2)
)

// Badge / indicator styles.
var (
	badgeOk = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1A202C")).
		Background(colorOk).
		Padding(0, 1).
		Bold(true)

	badgeWarn = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A202C")).
			Background(colorWarn).
			Padding(0, 1).
			Bold(true)

	badgeErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A202C")).
			Background(colorErr).
			Padding(0, 1).
			Bold(true)

	badgeAccent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A202C")).
			Background(colorAccent).
			Padding(0, 1).
			Bold(true)
)

// Metric value styles.
var (
	valueOk   = lipgloss.NewStyle().Foreground(colorOk).Bold(true)
	valueWarn  = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	valueErr   = lipgloss.NewStyle().Foreground(colorErr).Bold(true)
	valueNorm  = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

// cpuStyle picks the right color for a CPU percentage.
func cpuStyle(pct float64) lipgloss.Style {
	switch {
	case pct >= 80:
		return valueErr
	case pct >= 50:
		return valueWarn
	default:
		return valueOk
	}
}
