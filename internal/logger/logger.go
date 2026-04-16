package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ekorunov/watchctl/internal/model"
)

const dirName = "watchctl"

// DailyLog is the JSON structure written to disk — one file per day.
type DailyLog struct {
	Date   string            `json:"date"`
	Events []model.PeakEvent `json:"events"`
}

// Logger writes peak events to daily JSON files.
type Logger struct {
	dir      string
	maxFiles int
}

// New creates a Logger that writes to os.TempDir()/watchctl.
func New(maxFiles int) (*Logger, error) {
	dir := filepath.Join(os.TempDir(), dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	return &Logger{dir: dir, maxFiles: maxFiles}, nil
}

// Dir returns the log directory path.
func (l *Logger) Dir() string {
	return l.dir
}

// Run consumes peak events and appends them to daily log files.
func (l *Logger) Run(ctx context.Context, events <-chan model.PeakEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			_ = l.appendEvent(ev)
		}
	}
}

func (l *Logger) dailyPath(t time.Time) string {
	return filepath.Join(l.dir, t.Format("2006-01-02")+".json")
}

func (l *Logger) appendEvent(ev model.PeakEvent) error {
	path := l.dailyPath(ev.Timestamp)

	daily := l.loadDaily(path)
	daily.Events = append(daily.Events, ev)

	data, err := json.MarshalIndent(daily, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	l.rotate()
	return nil
}

func (l *Logger) loadDaily(path string) DailyLog {
	date := strings.TrimSuffix(filepath.Base(path), ".json")
	daily := DailyLog{Date: date}

	data, err := os.ReadFile(path)
	if err != nil {
		return daily
	}
	_ = json.Unmarshal(data, &daily)
	return daily
}

func (l *Logger) rotate() {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return
	}

	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			files = append(files, e)
		}
	}

	if len(files) <= l.maxFiles {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	excess := len(files) - l.maxFiles
	for i := 0; i < excess; i++ {
		_ = os.Remove(filepath.Join(l.dir, files[i].Name()))
	}
}

// LoadEvents reads all saved events from disk, sorted by time.
func (l *Logger) LoadEvents() ([]model.PeakEvent, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var events []model.PeakEvent
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(l.dir, e.Name()))
		if err != nil {
			continue
		}

		// Try daily format.
		var daily DailyLog
		if json.Unmarshal(data, &daily) == nil && len(daily.Events) > 0 {
			events = append(events, daily.Events...)
			continue
		}

		// Fallback: old single-event format.
		var ev model.PeakEvent
		if json.Unmarshal(data, &ev) == nil && !ev.Timestamp.IsZero() {
			events = append(events, ev)
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	return events, nil
}

// EventsSince returns events after the given time.
func (l *Logger) EventsSince(t time.Time) ([]model.PeakEvent, error) {
	all, err := l.LoadEvents()
	if err != nil {
		return nil, err
	}
	var result []model.PeakEvent
	for _, ev := range all {
		if ev.Timestamp.After(t) {
			result = append(result, ev)
		}
	}
	return result, nil
}
