package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ekorunov/watchctl/internal/model"
)

const dirName = "watchctl"

// Logger writes peak events to JSON files and handles rotation.
type Logger struct {
	dir         string
	maxFiles    int
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

// Run consumes peak events and writes each to a JSON file.
func (l *Logger) Run(ctx context.Context, events <-chan model.PeakEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			_ = l.write(ev)
			l.rotate()
		}
	}
}

func (l *Logger) write(ev model.PeakEvent) error {
	name := fmt.Sprintf("peak_%s_%s.json",
		ev.Timestamp.Format("20060102_150405"),
		ev.ID[:8],
	)
	path := filepath.Join(l.dir, name)

	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
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

// LoadEvents reads all saved peak events from disk, sorted by time.
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
		var ev model.PeakEvent
		if json.Unmarshal(data, &ev) == nil {
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
