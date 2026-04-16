package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ekorunov/watchctl/internal/analyzer"
	"github.com/ekorunov/watchctl/internal/collector"
	"github.com/ekorunov/watchctl/internal/history"
	"github.com/ekorunov/watchctl/internal/logger"
	"github.com/ekorunov/watchctl/internal/model"
	"github.com/ekorunov/watchctl/internal/ui"
)

func main() {
	cfg := model.DefaultConfig()

	log, err := logger.New(cfg.MaxLogFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	pastEvents, _ := log.LoadEvents()
	var currentNames []string

	// Channels.
	rawSnapCh := make(chan model.CPUSnapshot, 4)
	analyzerSnapCh := make(chan model.CPUSnapshot, 4)
	uiSnapCh := make(chan model.CPUSnapshot, 4)
	peakCh := make(chan model.PeakEvent, 8)
	logCh := make(chan model.PeakEvent, 8)
	uiPeakCh := make(chan model.PeakEvent, 8)
	threshCh := make(chan float64, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Collector.
	coll := collector.New(cfg.PollInterval, cfg.TopN)
	go coll.Run(ctx, rawSnapCh)

	// Fan-out: snapshots → analyzer + UI.
	go func() {
		for snap := range rawSnapCh {
			select {
			case analyzerSnapCh <- snap:
			default:
			}
			select {
			case uiSnapCh <- snap:
			default:
			}
		}
	}()

	// Analyzer.
	ana := analyzer.New(cfg.Threshold, cfg.DedupeWindow)
	go ana.Run(ctx, analyzerSnapCh, peakCh)

	// Threshold updates from UI → analyzer.
	go func() {
		for t := range threshCh {
			ana.SetThreshold(t)
		}
	}()

	// Fan-out: peaks → UI + logger.
	go func() {
		for ev := range peakCh {
			select {
			case uiPeakCh <- ev:
			default:
			}
			select {
			case logCh <- ev:
			default:
			}
		}
	}()

	// Logger.
	go log.Run(ctx, logCh)

	// History.
	summary := history.Analyze(pastEvents, currentNames)

	// TUI.
	m := ui.NewModel(uiSnapCh, uiPeakCh, cfg.Threshold, threshCh, log.Dir())

	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		p.Send(ui.HistoryLoadedMsg{
			Events:  pastEvents,
			Summary: summary,
		})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cancel()
}
