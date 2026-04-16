package model

import "time"

// CPUSnapshot represents a single point-in-time CPU measurement.
type CPUSnapshot struct {
	Timestamp  time.Time     `json:"timestamp"`
	TotalUsage float64       `json:"total_usage"`
	Processes  []ProcessInfo `json:"processes"`
}

// ProcessInfo holds per-process CPU details.
type ProcessInfo struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	Exe        string  `json:"exe,omitempty"`
	CPUPercent float64 `json:"cpu_percent"`
	CreateTime int64   `json:"create_time"`
}

// PeakEvent represents a detected CPU spike.
type PeakEvent struct {
	ID        string        `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	TotalCPU  float64       `json:"total_cpu"`
	Threshold float64       `json:"threshold"`
	TopProcs  []ProcessInfo `json:"top_procs"`
}

// HistorySummary provides aggregated analysis of past events.
type HistorySummary struct {
	TotalEvents   int
	FrequentProcs []ProcFrequency
	NewProcs      []string
	RecentEvents  []PeakEvent
}

// ProcFrequency tracks how often a process name appears in peaks.
type ProcFrequency struct {
	Name  string
	Count int
}

// PeakBurst groups consecutive peaks triggered by the same top process.
type PeakBurst struct {
	TopProcess string
	FirstTime  time.Time
	LastTime   time.Time
	MaxCPU     float64
	Count      int
	Events     []PeakEvent
}

// BurstGap is the maximum time between peaks to merge them into one burst.
const BurstGap = 5 * time.Minute

// GroupBursts merges consecutive PeakEvents with the same top process
// into bursts when they are within BurstGap of each other.
func GroupBursts(events []PeakEvent) []PeakBurst {
	if len(events) == 0 {
		return nil
	}

	var bursts []PeakBurst
	current := newBurst(events[0])

	for i := 1; i < len(events); i++ {
		ev := events[i]
		topName := topProcName(ev)
		gap := ev.Timestamp.Sub(current.LastTime)

		if topName == current.TopProcess && gap <= BurstGap {
			current.LastTime = ev.Timestamp
			current.Count++
			current.Events = append(current.Events, ev)
			if ev.TotalCPU > current.MaxCPU {
				current.MaxCPU = ev.TotalCPU
			}
		} else {
			bursts = append(bursts, current)
			current = newBurst(ev)
		}
	}
	bursts = append(bursts, current)
	return bursts
}

func newBurst(ev PeakEvent) PeakBurst {
	return PeakBurst{
		TopProcess: topProcName(ev),
		FirstTime:  ev.Timestamp,
		LastTime:   ev.Timestamp,
		MaxCPU:     ev.TotalCPU,
		Count:      1,
		Events:     []PeakEvent{ev},
	}
}

func topProcName(ev PeakEvent) string {
	if len(ev.TopProcs) > 0 {
		return ev.TopProcs[0].Name
	}
	return ""
}

// Config holds runtime configuration.
type Config struct {
	PollInterval time.Duration
	Threshold    float64
	TopN         int
	MaxLogFiles  int
	DedupeWindow time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PollInterval: 2 * time.Second,
		Threshold:    50.0,
		TopN:         10,
		MaxLogFiles:  50,
		DedupeWindow: 30 * time.Second,
	}
}
