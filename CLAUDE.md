# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**watchctl** — a Go TUI application for monitoring short-lived CPU spikes that are hard to catch manually. Uses an async goroutine+channel architecture with Bubble Tea for the interface.

## Build & Run

```bash
go build -o watchctl.exe ./cmd/watchctl/   # build binary
go install ./cmd/watchctl/                  # install to GOBIN
go build ./...                              # check all packages compile
go vet ./...                                # lint
go test ./...                               # run all tests
go test -run TestName ./internal/analyzer/  # single test
```

One-command launch: `go run ./cmd/watchctl/`

## Architecture

Four concurrent goroutines communicate via channels:

```
collector → [snapCh] → fan-out → [analyzerSnapCh] → analyzer → [peakCh] → fan-out → logger
                                  [uiSnapCh]     →  TUI                              → TUI
```

- **collector** (`internal/collector`) — polls CPU usage + top processes every 2s via gopsutil, sends `CPUSnapshot`
- **analyzer** (`internal/analyzer`) — detects peaks above threshold (50%), deduplicates similar events within 30s window using process-name fingerprinting
- **logger** (`internal/logger`) — writes peak events as JSON to `os.TempDir()/watchctl/`, rotates to keep last 50 files
- **TUI** (`internal/ui`) — Bubble Tea model on main thread, receives snapshots and peaks via channel-listening Cmds

Fan-out goroutines in `cmd/watchctl/main.go` distribute snapshots and peaks to multiple consumers with non-blocking sends.

## TUI Design System

Uses the Charmbracelet ecosystem (Bubble Tea + Lipgloss + Bubbles) with a shared color palette:

| Role       | Color     |
|------------|-----------|
| Accent     | `#5F9FFF` |
| OK/Green   | `#68D391` |
| Warn       | `#F6C950` |
| Error      | `#FC8181` |
| Muted      | `#A0AEC0` |
| Border     | `#4A5568` |
| Select bg  | `#2C5282` |

All style definitions live in `internal/ui/styles.go`. Follow existing patterns when adding new styled elements.

Three screens: **Live** (CPU gauge + process table), **History** (event list + summary panel), **Details** (full event snapshot). Navigation via Tab, arrows/j/k, Enter, Esc, q.

## Key Types

- `model.CPUSnapshot` — point-in-time CPU + process list
- `model.PeakEvent` — detected spike with ID, threshold, top processes
- `model.HistorySummary` — aggregated analysis (frequent procs, new procs)
- `model.Config` — runtime settings (poll interval, threshold, topN, max log files, dedupe window)

## Extending

To add new metrics (memory, disk, network):
1. Add fields to `CPUSnapshot` (or create a parallel snapshot type)
2. Extend `collector.collect()` with new gopsutil calls
3. Add display in `internal/ui/view.go` render methods

Log files: `os.TempDir()/watchctl/peak_*.json`
