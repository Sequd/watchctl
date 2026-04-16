package ui

import (
	"github.com/ekorunov/watchctl/internal/model"
)

// SnapshotMsg carries a fresh CPU snapshot to the UI.
type SnapshotMsg struct {
	Snapshot model.CPUSnapshot
}

// PeakMsg carries a newly detected peak event to the UI.
type PeakMsg struct {
	Event model.PeakEvent
}

// HistoryLoadedMsg carries the history summary loaded at startup.
type HistoryLoadedMsg struct {
	Events  []model.PeakEvent
	Summary model.HistorySummary
}

// tickMsg triggers periodic UI refresh.
type tickMsg struct{}
