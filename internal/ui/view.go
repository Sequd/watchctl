package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ekorunov/watchctl/internal/model"
)

// Braille dot bit values, ordered bottom-to-top for each column.
var (
	brailleBase = rune(0x2800)
	leftDots    = [4]rune{0x40, 0x04, 0x02, 0x01} // dots 7,3,2,1
	rightDots   = [4]rune{0x80, 0x20, 0x10, 0x08}  // dots 8,6,5,4
)

func (m Model) renderView() string {
	header := m.renderHeader()
	tabs := m.renderTabs()

	var body string
	switch m.activeTab {
	case TabLive:
		body = m.renderLive()
	case TabHistory:
		body = m.renderHistory()
	case TabDetails:
		body = m.renderDetails()
	}

	statusBar := m.renderStatusBar()
	helpBar := m.renderHelp()

	parts := []string{header, tabs, "", body}
	if statusBar != "" {
		parts = append(parts, "", statusBar)
	}
	parts = append(parts, "", helpBar)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderHeader() string {
	title := titleStyle.Render("watchctl")
	sub := subtitleStyle.Render("CPU Activity Monitor")
	return lipgloss.JoinHorizontal(lipgloss.Bottom, title, "  ", sub)
}

func (m Model) renderTabs() string {
	var tabs []string
	for i, name := range tabNames {
		if Tab(i) == TabDetails && m.activeTab != TabDetails {
			continue
		}
		if Tab(i) == m.activeTab {
			tabs = append(tabs, tabActive.Render("● "+name))
		} else {
			tabs = append(tabs, tabInactive.Render("  "+name))
		}
	}
	sep := separatorStyle.Render(" │ ")
	return strings.Join(tabs, sep)
}

// --- Live screen ---

func (m Model) renderLive() string {
	leftCol := m.renderCPUPanel()
	rightCol := m.renderProcessTable(m.snapshot.Processes, true)

	sep := separatorStyle.Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftCol,
		" "+sep+" ",
		rightCol,
	)
}

func (m Model) renderCPUPanel() string {
	pct := m.snapshot.TotalUsage
	style := cpuStyle(pct)

	label := labelStyle.Render("CPU Total")
	value := style.Render(fmt.Sprintf("%.1f%%", pct))
	bar := renderBar(pct, 22)

	lines := []string{
		label,
		value + "  " + bar,
	}

	// Braille chart.
	if len(m.cpuHistory) > 1 {
		lines = append(lines, "")
		chart := renderBrailleChart(m.cpuHistory, 14, m.threshold)
		stats := chartStats(m.cpuHistory)
		lines = append(lines, chart)
		lines = append(lines, labelStyle.Render(stats))
	}

	// Threshold.
	lines = append(lines, "")
	threshLabel := fmt.Sprintf("  %s %.0f%%",
		labelStyle.Render("Threshold:"),
		m.threshold,
	)
	lines = append(lines, threshLabel)
	lines = append(lines, helpStyle.Render("  [/] adjust"))

	// Peak indicator.
	if time.Since(m.lastPeakTime) < 5*time.Second {
		lines = append(lines, "", badgeErr.Render(" PEAK "))
	}

	if m.peakCount > 0 {
		lines = append(lines, fmt.Sprintf("  %s %d",
			labelStyle.Render("Peaks:"),
			m.peakCount,
		))
	}

	width := 32
	return lipgloss.NewStyle().Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func renderBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	barColor := barColorForPct(pct)
	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5568"))

	return filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))
}

func barColorForPct(pct float64) lipgloss.Color {
	switch {
	case pct >= 80:
		return lipgloss.Color("#FC8181")
	case pct >= 50:
		return lipgloss.Color("#F6C950")
	default:
		return lipgloss.Color("#68D391")
	}
}

// renderBrailleChart draws a 2-row braille area chart.
// Each character holds 2 data points horizontally and 4 levels vertically per row (8 total).
func renderBrailleChart(data []float64, widthChars int, threshold float64) string {
	maxPoints := widthChars * 2
	// Take last maxPoints samples.
	if len(data) > maxPoints {
		data = data[len(data)-maxPoints:]
	}

	// Pad left with zeros if not enough data.
	if len(data) < maxPoints {
		padded := make([]float64, maxPoints)
		copy(padded[maxPoints-len(data):], data)
		data = padded
	}

	maxVal := 100.0 // always scale to 100%

	var topRow, bottomRow strings.Builder

	for i := 0; i < len(data); i += 2 {
		lv := data[i]
		rv := 0.0
		if i+1 < len(data) {
			rv = data[i+1]
		}

		// Map to 0-8 levels.
		ll := int(math.Round(lv / maxVal * 8))
		rl := int(math.Round(rv / maxVal * 8))
		if ll > 8 {
			ll = 8
		}
		if rl > 8 {
			rl = 8
		}
		if ll < 0 {
			ll = 0
		}
		if rl < 0 {
			rl = 0
		}

		// Top row: levels 5-8, Bottom row: levels 1-4.
		topChar := brailleBase
		bottomChar := brailleBase

		// Left column.
		topLeft := ll - 4
		if topLeft < 0 {
			topLeft = 0
		}
		bottomLeft := ll
		if bottomLeft > 4 {
			bottomLeft = 4
		}
		for j := 0; j < topLeft; j++ {
			topChar |= leftDots[j]
		}
		for j := 0; j < bottomLeft; j++ {
			bottomChar |= leftDots[j]
		}

		// Right column.
		topRight := rl - 4
		if topRight < 0 {
			topRight = 0
		}
		bottomRight := rl
		if bottomRight > 4 {
			bottomRight = 4
		}
		for j := 0; j < topRight; j++ {
			topChar |= rightDots[j]
		}
		for j := 0; j < bottomRight; j++ {
			bottomChar |= rightDots[j]
		}

		// Color by max of two values.
		mv := lv
		if rv > mv {
			mv = rv
		}

		ts := string(topChar)
		bs := string(bottomChar)

		c := colorForPct(mv)
		topRow.WriteString(lipgloss.NewStyle().Foreground(c).Render(ts))
		bottomRow.WriteString(lipgloss.NewStyle().Foreground(c).Render(bs))
	}

	return topRow.String() + "\n" + bottomRow.String()
}

func colorForPct(pct float64) lipgloss.Color {
	switch {
	case pct >= 80:
		return lipgloss.Color("#FC8181")
	case pct >= 50:
		return lipgloss.Color("#F6C950")
	default:
		return lipgloss.Color("#68D391")
	}
}

func chartStats(data []float64) string {
	if len(data) == 0 {
		return ""
	}
	min, max, sum := data[0], data[0], 0.0
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	avg := sum / float64(len(data))
	return fmt.Sprintf("  min %.0f%%  avg %.0f%%  max %.0f%%", min, avg, max)
}

// --- Process table with Live and Peak columns ---

func (m Model) renderProcessTable(procs []model.ProcessInfo, interactive bool) string {
	if len(procs) == 0 {
		return rowDimStyle.Render("No process data yet...")
	}

	hdr := headerStyle.Render(
		fmt.Sprintf("%-7s %-22s %7s %7s", "PID", "Name", "CPU%", "Peak%"),
	)

	var rows []string
	rows = append(rows, hdr)

	for i, p := range procs {
		name := p.Name
		if len(name) > 22 {
			name = name[:19] + "..."
		}

		cpuStr := cpuStyle(p.CPUPercent).Render(fmt.Sprintf("%6.1f%%", p.CPUPercent))

		peak := m.procPeaks[p.PID]
		peakStr := lipgloss.NewStyle().Foreground(colorFaint).Render(fmt.Sprintf("%6.1f%%", peak))

		line := fmt.Sprintf("%-7d %-22s %s %s", p.PID, name, cpuStr, peakStr)

		if interactive && i == m.cursor {
			rows = append(rows, rowSelectedStyle.Render(line))
		} else {
			rows = append(rows, rowNormalStyle.Render(line))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// --- History screen (burst-aggregated) ---

func (m Model) renderHistory() string {
	if len(m.bursts) == 0 {
		return rowDimStyle.Render("No peak events recorded yet.")
	}

	hdr := headerStyle.Render(
		fmt.Sprintf("%-25s %7s %5s  %-22s", "Time", "Peak%", "Hits", "Process"),
	)

	var rows []string
	rows = append(rows, hdr)

	bursts := reverseBursts(m.bursts)
	for i, b := range bursts {
		name := b.TopProcess
		if len(name) > 22 {
			name = name[:19] + "..."
		}

		ts := b.LastTime.Format("01-02 15:04:05")

		cpuStr := cpuStyle(b.MaxCPU).Render(fmt.Sprintf("%6.1f%%", b.MaxCPU))
		countStr := fmt.Sprintf("%3dx", b.Count)
		if b.Count == 1 {
			countStr = "   1"
		}

		line := fmt.Sprintf("%-25s %s %s  %-22s", ts, cpuStr, countStr, name)

		if i == m.cursor {
			rows = append(rows, rowSelectedStyle.Render(line))
		} else {
			rows = append(rows, rowNormalStyle.Render(line))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) renderSummaryPanel() string {
	var lines []string
	lines = append(lines, headerStyle.Render("Summary"))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("  %s %d",
		labelStyle.Render("Total peaks:"),
		len(m.peakEvents),
	))

	if len(m.summary.FrequentProcs) > 0 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("  Frequent:"))
		n := 5
		if n > len(m.summary.FrequentProcs) {
			n = len(m.summary.FrequentProcs)
		}
		for _, f := range m.summary.FrequentProcs[:n] {
			lines = append(lines, fmt.Sprintf("    %s (%dx)", f.Name, f.Count))
		}
	}

	if len(m.summary.NewProcs) > 0 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("  New in peaks:"))
		n := 5
		if n > len(m.summary.NewProcs) {
			n = len(m.summary.NewProcs)
		}
		for _, name := range m.summary.NewProcs[:n] {
			lines = append(lines, fmt.Sprintf("    %s", name))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// --- Details screen (burst events) ---

func (m Model) renderDetails() string {
	bursts := reverseBursts(m.bursts)
	if m.detailIdx >= len(bursts) {
		return rowDimStyle.Render("No event selected.")
	}
	b := bursts[m.detailIdx]

	// Burst header.
	var timeRange string
	if b.Count == 1 {
		timeRange = b.FirstTime.Format("2006-01-02 15:04:05")
	} else {
		timeRange = b.FirstTime.Format("2006-01-02 15:04:05") + " — " + b.LastTime.Format("15:04:05")
	}

	hdr := []string{
		headerStyle.Render("Burst Details"),
		"",
		fmt.Sprintf("  %s %s",
			labelStyle.Render("Process:"),
			lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(b.TopProcess),
		),
		fmt.Sprintf("  %s %s",
			labelStyle.Render("Time:"),
			timeRange,
		),
		fmt.Sprintf("  %s %s",
			labelStyle.Render("Peak CPU:"),
			cpuStyle(b.MaxCPU).Render(fmt.Sprintf("%.1f%%", b.MaxCPU)),
		),
		fmt.Sprintf("  %s %d",
			labelStyle.Render("Events:"),
			b.Count,
		),
		"",
		headerStyle.Render("Events in burst"),
	}

	header := lipgloss.JoinVertical(lipgloss.Left, hdr...)

	// Events table.
	evHdr := headerStyle.Render(
		fmt.Sprintf("  %-20s %8s  %-25s", "Time", "CPU%", "Top Process"),
	)
	var rows []string
	rows = append(rows, evHdr)

	for i, ev := range b.Events {
		topName := ""
		if len(ev.TopProcs) > 0 {
			topName = ev.TopProcs[0].Name
			if len(topName) > 25 {
				topName = topName[:22] + "..."
			}
		}

		ts := ev.Timestamp.Format("15:04:05")
		cpuStr := cpuStyle(ev.TotalCPU).Render(fmt.Sprintf("%7.1f%%", ev.TotalCPU))
		line := fmt.Sprintf("  %-20s %s  %-25s", ts, cpuStr, topName)

		if i == m.cursor {
			rows = append(rows, rowSelectedStyle.Render(line))
		} else {
			rows = append(rows, rowNormalStyle.Render(line))
		}
	}

	table := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return lipgloss.JoinVertical(lipgloss.Left, header, table)
}

// --- Status & Help ---

func (m Model) renderStatusBar() string {
	if m.statusMsg == "" || time.Now().After(m.statusExpiry) {
		return ""
	}
	return statusBarStyle.Render("▸ " + m.statusMsg)
}

func (m Model) renderHelp() string {
	var parts []string
	switch m.activeTab {
	case TabLive:
		parts = []string{"↑/k ↓/j nav", "[/] threshold", "r reset", "tab switch", "q quit"}
	case TabHistory:
		parts = []string{"↑/k ↓/j nav", "enter details", "tab switch", "q quit"}
	case TabDetails:
		parts = []string{"↑/k ↓/j nav", "esc back", "tab switch", "q quit"}
	}

	styled := make([]string, len(parts))
	for i, p := range parts {
		styled[i] = helpStyle.Render(p)
	}
	sep := separatorStyle.Render(" │ ")
	return strings.Join(styled, sep)
}

// --- Helpers ---

func reverseEvents(events []model.PeakEvent) []model.PeakEvent {
	n := len(events)
	rev := make([]model.PeakEvent, n)
	for i, ev := range events {
		rev[n-1-i] = ev
	}
	return rev
}
