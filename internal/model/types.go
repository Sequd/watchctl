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
