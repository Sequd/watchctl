package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ekorunov/watchctl/internal/model"
)

// Tab represents the active screen.
type Tab int

const (
	TabLive Tab = iota
	TabHistory
	TabAnalysis
	TabDetails
)

const cpuHistoryLen = 60

var tabNames = []string{"Live", "History", "Analysis", "Details"}

// Model is the top-level Bubble Tea model.
type Model struct {
	// Data
	snapshot     model.CPUSnapshot
	peakEvents   []model.PeakEvent
	bursts       []model.PeakBurst
	summary      model.HistorySummary
	lastPeakTime time.Time
	peakCount    int
	cpuHistory   []float64
	procPeaks    map[int32]float64 // PID → max CPU% seen

	// Threshold
	threshold float64
	threshCh  chan<- float64

	// Log directory
	logDir string

	// UI state
	activeTab    Tab
	cursor       int
	detailIdx    int
	width        int
	height       int
	statusMsg    string
	statusExpiry time.Time

	// Channels (read in Cmds)
	snapCh <-chan model.CPUSnapshot
	peakCh <-chan model.PeakEvent
}

// NewModel creates the initial UI model.
func NewModel(snapCh <-chan model.CPUSnapshot, peakCh <-chan model.PeakEvent, threshold float64, threshCh chan<- float64, logDir string) Model {
	return Model{
		snapCh:    snapCh,
		peakCh:    peakCh,
		threshold: threshold,
		threshCh:  threshCh,
		logDir:    logDir,
		procPeaks: make(map[int32]float64),
		width:     120,
		height:    30,
	}
}

// Init starts the background listeners.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		listenSnapshot(m.snapCh),
		listenPeak(m.peakCh),
		tickCmd(),
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case SnapshotMsg:
		m.snapshot = msg.Snapshot
		m.pushCPU(msg.Snapshot.TotalUsage)
		m.updatePeaks(msg.Snapshot.Processes)
		return m, listenSnapshot(m.snapCh)

	case PeakMsg:
		m.peakEvents = append(m.peakEvents, msg.Event)
		m.rebuildBursts()
		m.lastPeakTime = msg.Event.Timestamp
		m.peakCount++
		m.setStatus("Peak detected: %.1f%% CPU", msg.Event.TotalCPU)
		return m, listenPeak(m.peakCh)

	case HistoryLoadedMsg:
		m.peakEvents = msg.Events
		m.rebuildBursts()
		m.summary = msg.Summary
		if len(msg.Events) > 0 {
			m.setStatus("Loaded %d historical events", len(msg.Events))
		}
		return m, nil

	case tickMsg:
		return m, tickCmd()
	}

	return m, nil
}

func (m *Model) pushCPU(pct float64) {
	m.cpuHistory = append(m.cpuHistory, pct)
	if len(m.cpuHistory) > cpuHistoryLen {
		m.cpuHistory = m.cpuHistory[len(m.cpuHistory)-cpuHistoryLen:]
	}
}

func (m *Model) updatePeaks(procs []model.ProcessInfo) {
	for _, p := range procs {
		if p.CPUPercent > m.procPeaks[p.PID] {
			m.procPeaks[p.PID] = p.CPUPercent
		}
	}
}

func (m *Model) rebuildBursts() {
	m.bursts = model.GroupBursts(m.peakEvents)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		if m.activeTab == TabDetails {
			m.activeTab = TabHistory
			return m, nil
		}
		return m, tea.Quit

	case key.Matches(msg, keys.Tab):
		switch m.activeTab {
		case TabLive:
			m.activeTab = TabHistory
		case TabHistory:
			m.activeTab = TabAnalysis
		case TabAnalysis:
			m.activeTab = TabLive
		case TabDetails:
			m.activeTab = TabLive
		}
		m.cursor = 0
		return m, nil

	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		m.cursor++
		m.clampCursor()
		return m, nil

	case key.Matches(msg, keys.Enter):
		if m.activeTab == TabHistory && len(m.bursts) > 0 {
			bursts := reverseBursts(m.bursts)
			if m.cursor < len(bursts) {
				m.detailIdx = m.cursor
				m.activeTab = TabDetails
				m.cursor = 0
			}
			return m, nil
		}
		return m, nil

	case key.Matches(msg, keys.Back):
		if m.activeTab == TabDetails {
			m.activeTab = TabHistory
			m.cursor = m.detailIdx
			return m, nil
		}
		return m, nil

	case key.Matches(msg, keys.Reset):
		if m.activeTab == TabLive {
			m.snapshot = model.CPUSnapshot{}
			m.cpuHistory = nil
			m.procPeaks = make(map[int32]float64)
			m.lastPeakTime = time.Time{}
			m.peakCount = 0
			m.summary = model.HistorySummary{}
			m.cursor = 0
			m.setStatus("Statistics reset")
			return m, nil
		}
		return m, nil

	case key.Matches(msg, keys.ThreshUp):
		if m.activeTab == TabLive && m.threshold < 95 {
			m.threshold += 5
			select {
			case m.threshCh <- m.threshold:
			default:
			}
			m.setStatus("Threshold: %.0f%%", m.threshold)
		}
		return m, nil

	case key.Matches(msg, keys.ThreshDn):
		if m.activeTab == TabLive && m.threshold > 10 {
			m.threshold -= 5
			select {
			case m.threshCh <- m.threshold:
			default:
			}
			m.setStatus("Threshold: %.0f%%", m.threshold)
		}
		return m, nil

	case key.Matches(msg, keys.OpenDir):
		openPath(m.logDir)
		m.setStatus("Opened log directory")
		return m, nil

	case key.Matches(msg, keys.OpenLog):
		if f := lastLogFile(m.logDir); f != "" {
			openInCode(f)
			m.setStatus("Opened %s", filepath.Base(f))
		} else {
			m.setStatus("No log files yet")
		}
		return m, nil
	}

	return m, nil
}

func openPath(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}

func openInCode(path string) {
	cmd := exec.Command("code", path)
	_ = cmd.Start()
}

func lastLogFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var jsons []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			jsons = append(jsons, e.Name())
		}
	}
	if len(jsons) == 0 {
		return ""
	}
	sort.Strings(jsons)
	return filepath.Join(dir, jsons[len(jsons)-1])
}

func (m *Model) clampCursor() {
	max := 0
	switch m.activeTab {
	case TabLive:
		max = len(m.snapshot.Processes) - 1
	case TabHistory:
		max = len(m.bursts) - 1
	case TabDetails:
		bursts := reverseBursts(m.bursts)
		if m.detailIdx < len(bursts) {
			max = len(bursts[m.detailIdx].Events) - 1
		}
	}
	if max < 0 {
		max = 0
	}
	if m.cursor > max {
		m.cursor = max
	}
}

func (m *Model) setStatus(format string, args ...interface{}) {
	m.statusMsg = fmt.Sprintf(format, args...)
	m.statusExpiry = time.Now().Add(3 * time.Second)
}

// View renders the full screen.
func (m Model) View() string {
	return m.renderView()
}

// --- tea.Cmd helpers ---

func listenSnapshot(ch <-chan model.CPUSnapshot) tea.Cmd {
	return func() tea.Msg {
		snap, ok := <-ch
		if !ok {
			return nil
		}
		return SnapshotMsg{Snapshot: snap}
	}
}

func listenPeak(ch <-chan model.PeakEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return PeakMsg{Event: ev}
	}
}

func reverseBursts(bursts []model.PeakBurst) []model.PeakBurst {
	n := len(bursts)
	rev := make([]model.PeakBurst, n)
	for i, b := range bursts {
		rev[n-1-i] = b
	}
	return rev
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
