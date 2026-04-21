package ui

import (
	"fmt"
	"math"
	"sort"
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
	statusBar := m.renderStatusBar()
	helpBar := m.renderHelp()

	var body string
	switch m.activeTab {
	case TabLive:
		body = m.renderLive()
	case TabHistory:
		body = m.renderHistory()
	case TabAnalysis:
		body = m.renderAnalysis()
	case TabDetails:
		body = m.renderDetails()
	}

	// Fixed lines: header(1) + tabs(1) + blank(1) + blank(1) + helpBar(1) = 5
	// Plus optional statusBar: blank(1) + statusBar(1) = 2
	fixedLines := 5
	if statusBar != "" {
		fixedLines += 2
	}
	bodyHeight := m.height - fixedLines
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Pad body with blank lines so helpBar is always pinned to the bottom.
	bodyLines := strings.Count(body, "\n") + 1
	if bodyLines < bodyHeight {
		body += strings.Repeat("\n", bodyHeight-bodyLines)
	}

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
		tab := Tab(i)
		if tab == TabDetails && m.activeTab != TabDetails {
			continue
		}
		if tab == m.activeTab {
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
	chart := m.renderCPUChartFull()

	stats := m.renderCPUSummary()
	procs := m.renderProcessTable(m.snapshot.Processes, true)

	sep := separatorStyle.Render("│")
	bottom := lipgloss.JoinHorizontal(lipgloss.Top,
		stats,
		" "+sep+" ",
		procs,
	)

	return lipgloss.JoinVertical(lipgloss.Left, chart, "", bottom)
}

func (m Model) renderCPUChartFull() string {
	pct := m.snapshot.TotalUsage

	// Chart occupies half the terminal width; border(2) + padding(2) = 4 overhead.
	chartWidth := m.width/2 - 4
	if chartWidth < 10 {
		chartWidth = 10
	}

	barWidth := m.width/2 - 33
	if barWidth < 6 {
		barWidth = 6
	}

	// Header line — no border, just above the chart.
	header := lipgloss.JoinHorizontal(lipgloss.Bottom,
		labelStyle.Render("CPU "),
		cpuStyle(pct).Render(fmt.Sprintf("%.1f%%", pct)),
		"  ",
		renderBar(pct, barWidth),
		labelStyle.Render(fmt.Sprintf("  %.0f%% threshold", m.threshold)),
		helpStyle.Render("  [/]"),
	)

	lines := []string{header}

	if len(m.cpuHistory) > 1 {
		chartContent := renderBrailleChart(m.cpuHistory, chartWidth, 4)
		lines = append(lines, manualBorder(chartContent, colorBorder))
		lines = append(lines, labelStyle.Render(chartStats(m.cpuHistory)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// manualBorder draws a rounded border around pre-rendered ANSI content line by line,
// bypassing lipgloss reflow which mis-measures ANSI sequences and causes wrapping artifacts.
func manualBorder(content string, color lipgloss.Color) string {
	bc := lipgloss.NewStyle().Foreground(color)
	rows := strings.Split(content, "\n")

	// Measure visual width from the first non-empty row.
	innerWidth := 0
	for _, r := range rows {
		if w := lipgloss.Width(r); w > innerWidth {
			innerWidth = w
		}
	}
	dash := strings.Repeat("─", innerWidth+2)

	out := make([]string, 0, len(rows)+2)
	out = append(out, bc.Render("╭"+dash+"╮"))
	for _, r := range rows {
		out = append(out, bc.Render("│")+" "+r+" "+bc.Render("│"))
	}
	out = append(out, bc.Render("╰"+dash+"╯"))
	return strings.Join(out, "\n")
}

func (m Model) renderCPUSummary() string {
	var lines []string

	if time.Since(m.lastPeakTime) < 5*time.Second {
		lines = append(lines, " "+badgeErr.Render(" PEAK "))
	}

	if m.peakCount > 0 {
		lines = append(lines, fmt.Sprintf("  %s %d",
			labelStyle.Render("Peaks:"),
			m.peakCount,
		))
	} else {
		lines = append(lines, rowDimStyle.Render("No peaks yet"))
	}

	return lipgloss.NewStyle().Width(28).Render(
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

// renderBrailleChart draws a numRows-row braille area chart.
// Each character holds 2 data points horizontally and 4 levels vertically (numRows*4 total levels).
func renderBrailleChart(data []float64, widthChars int, numRows int) string {
	maxPoints := widthChars * 2
	if len(data) > maxPoints {
		data = data[len(data)-maxPoints:]
	}
	if len(data) < maxPoints {
		padded := make([]float64, maxPoints)
		copy(padded[maxPoints-len(data):], data)
		data = padded
	}

	totalLevels := numRows * 4
	rows := make([]strings.Builder, numRows)

	for i := 0; i < len(data); i += 2 {
		lv := data[i]
		rv := 0.0
		if i+1 < len(data) {
			rv = data[i+1]
		}

		ll := int(math.Round(lv / 100.0 * float64(totalLevels)))
		rl := int(math.Round(rv / 100.0 * float64(totalLevels)))
		if ll > totalLevels {
			ll = totalLevels
		}
		if ll < 0 {
			ll = 0
		}
		if rl > totalLevels {
			rl = totalLevels
		}
		if rl < 0 {
			rl = 0
		}

		mv := lv
		if rv > mv {
			mv = rv
		}
		st := lipgloss.NewStyle().Foreground(colorForPct(mv))

		for row := 0; row < numRows; row++ {
			// row 0 = top (highest levels), row numRows-1 = bottom (lowest levels).
			base := (numRows - 1 - row) * 4

			leftFill := ll - base
			if leftFill < 0 {
				leftFill = 0
			}
			if leftFill > 4 {
				leftFill = 4
			}
			rightFill := rl - base
			if rightFill < 0 {
				rightFill = 0
			}
			if rightFill > 4 {
				rightFill = 4
			}

			ch := brailleBase
			for j := 0; j < leftFill; j++ {
				ch |= leftDots[j]
			}
			for j := 0; j < rightFill; j++ {
				ch |= rightDots[j]
			}
			rows[row].WriteString(st.Render(string(ch)))
		}
	}

	lines := make([]string, numRows)
	for i, r := range rows {
		lines[i] = r.String()
	}
	return strings.Join(lines, "\n")
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

	bursts := reverseBursts(m.bursts)

	visible := m.height - 8
	if visible < 1 {
		visible = 1
	}
	offset := m.historyOffset
	end := offset + visible
	if end > len(bursts) {
		end = len(bursts)
	}

	var rows []string
	rows = append(rows, hdr)

	for i, b := range bursts[offset:end] {
		actualIdx := offset + i
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

		if actualIdx == m.cursor {
			rows = append(rows, rowSelectedStyle.Render(line))
		} else {
			rows = append(rows, rowNormalStyle.Render(line))
		}
	}

	if len(bursts) > visible {
		scrollInfo := fmt.Sprintf("  %d–%d / %d", offset+1, end, len(bursts))
		rows = append(rows, rowDimStyle.Render(scrollInfo))
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

// --- Analysis screen ---

func (m Model) renderAnalysis() string {
	if len(m.peakEvents) == 0 {
		return rowDimStyle.Render("No data for analysis yet. Peaks will appear here when CPU exceeds threshold.")
	}

	offenders := m.renderTopOffenders()
	hourly := m.renderHourlyActivity()
	daily := m.renderDailyTrend()

	leftCol := lipgloss.JoinVertical(lipgloss.Left, offenders, "", hourly)
	sep := separatorStyle.Render("│")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftCol,
		" "+sep+" ",
		daily,
	)
}

// renderTopOffenders shows a ranked list of processes by peak count.
func (m Model) renderTopOffenders() string {
	// Count appearances per process name.
	type offender struct {
		Name   string
		Count  int
		MaxCPU float64
	}
	counts := make(map[string]*offender)
	for _, ev := range m.peakEvents {
		if len(ev.TopProcs) == 0 {
			continue
		}
		name := ev.TopProcs[0].Name
		o, ok := counts[name]
		if !ok {
			o = &offender{Name: name}
			counts[name] = o
		}
		o.Count++
		if ev.TotalCPU > o.MaxCPU {
			o.MaxCPU = ev.TotalCPU
		}
	}

	// Sort by count descending.
	var list []offender
	for _, o := range counts {
		list = append(list, *o)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Count > list[j].Count
	})

	n := 8
	if n > len(list) {
		n = len(list)
	}
	list = list[:n]

	maxCount := 0
	for _, o := range list {
		if o.Count > maxCount {
			maxCount = o.Count
		}
	}

	lines := []string{headerStyle.Render("Top Offenders"), ""}

	barWidth := 20
	for _, o := range list {
		name := o.Name
		if len(name) > 22 {
			name = name[:19] + "..."
		}

		filled := 0
		if maxCount > 0 {
			filled = o.Count * barWidth / maxCount
		}
		if filled < 1 && o.Count > 0 {
			filled = 1
		}

		bar := lipgloss.NewStyle().Foreground(colorAccent).Render(strings.Repeat("█", filled)) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#2D3748")).Render(strings.Repeat("░", barWidth-filled))

		cpuStr := cpuStyle(o.MaxCPU).Render(fmt.Sprintf("%.0f%%", o.MaxCPU))

		lines = append(lines, fmt.Sprintf("  %-22s %3d  %s  %s", name, o.Count, bar, cpuStr))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderHourlyActivity shows a 24-hour heatmap of peak frequency.
func (m Model) renderHourlyActivity() string {
	var hours [24]int
	for _, ev := range m.peakEvents {
		h := ev.Timestamp.Hour()
		hours[h]++
	}

	maxH := 0
	for _, c := range hours {
		if c > maxH {
			maxH = c
		}
	}

	blocks := []rune{'░', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	lines := []string{headerStyle.Render("Hourly Activity"), ""}

	// Hour labels.
	var labelRow strings.Builder
	labelRow.WriteString("  ")
	for h := 0; h < 24; h++ {
		labelRow.WriteString(fmt.Sprintf("%-3d", h))
	}
	lines = append(lines, labelStyle.Render(labelRow.String()))

	// Bar row.
	var barRow strings.Builder
	barRow.WriteString("  ")
	for h := 0; h < 24; h++ {
		idx := 0
		if maxH > 0 && hours[h] > 0 {
			idx = hours[h] * (len(blocks) - 1) / maxH
			if idx < 1 {
				idx = 1
			}
		}
		ch := string(blocks[idx])
		c := colorForPct(float64(hours[h]) / float64(max(maxH, 1)) * 100)
		barRow.WriteString(lipgloss.NewStyle().Foreground(c).Render(ch))
		barRow.WriteString("  ")
	}
	lines = append(lines, barRow.String())

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderDailyTrend shows peaks per day for the last 7 days.
func (m Model) renderDailyTrend() string {
	now := time.Now()
	days := 7
	type dayEntry struct {
		Date  time.Time
		Count int
	}
	entries := make([]dayEntry, days)
	for i := 0; i < days; i++ {
		entries[days-1-i] = dayEntry{Date: now.AddDate(0, 0, -i)}
	}

	// Count events per day.
	for _, ev := range m.peakEvents {
		evDay := ev.Timestamp.Format("2006-01-02")
		for i := range entries {
			if entries[i].Date.Format("2006-01-02") == evDay {
				entries[i].Count++
			}
		}
	}

	maxD := 0
	for _, e := range entries {
		if e.Count > maxD {
			maxD = e.Count
		}
	}

	blocks := []rune{'░', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	barWidth := 15

	lines := []string{headerStyle.Render("Daily Trend (7 days)"), ""}

	for _, e := range entries {
		dayLabel := e.Date.Format("Jan 02")
		weekday := e.Date.Format("Mon")

		filled := 0
		if maxD > 0 && e.Count > 0 {
			filled = e.Count * barWidth / maxD
			if filled < 1 {
				filled = 1
			}
		}

		idx := 0
		if maxD > 0 && e.Count > 0 {
			idx = e.Count * (len(blocks) - 1) / maxD
			if idx < 1 {
				idx = 1
			}
		}

		c := colorForPct(float64(e.Count) / float64(max(maxD, 1)) * 100)
		bar := lipgloss.NewStyle().Foreground(c).Render(strings.Repeat(string(blocks[idx]), filled)) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#2D3748")).Render(strings.Repeat("░", barWidth-filled))

		countStr := fmt.Sprintf("%d", e.Count)
		if e.Count == 0 {
			countStr = lipgloss.NewStyle().Foreground(colorFaint).Render("0")
		}

		lines = append(lines, fmt.Sprintf("  %s %-3s  %s  %s peaks",
			labelStyle.Render(dayLabel),
			labelStyle.Render(weekday),
			bar,
			countStr,
		))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
		parts = []string{"↑/k ↓/j nav", "[/] threshold", "r reset", "o logs", "l last log", "tab switch", "q quit"}
	case TabHistory:
		parts = []string{"↑/k ↓/j nav", "enter details", "o logs", "l last log", "tab switch", "q quit"}
	case TabAnalysis:
		parts = []string{"o logs", "l last log", "tab switch", "q quit"}
	case TabDetails:
		parts = []string{"↑/k ↓/j nav", "esc back", "o logs", "l last log", "tab switch", "q quit"}
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
