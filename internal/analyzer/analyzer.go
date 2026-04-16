package analyzer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ekorunov/watchctl/internal/model"
)

// Analyzer detects CPU peaks from snapshots and emits deduplicated events.
type Analyzer struct {
	mu           sync.Mutex
	threshold    float64
	dedupeWindow time.Duration
	lastEvent    *model.PeakEvent
}

// New creates an Analyzer with the given threshold and dedup window.
func New(threshold float64, dedupeWindow time.Duration) *Analyzer {
	return &Analyzer{
		threshold:    threshold,
		dedupeWindow: dedupeWindow,
	}
}

// SetThreshold updates the peak detection threshold (thread-safe).
func (a *Analyzer) SetThreshold(t float64) {
	a.mu.Lock()
	a.threshold = t
	a.mu.Unlock()
}

// Threshold returns the current threshold (thread-safe).
func (a *Analyzer) Threshold() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.threshold
}

// Run reads snapshots from in, and sends detected peak events to out.
func (a *Analyzer) Run(ctx context.Context, in <-chan model.CPUSnapshot, out chan<- model.PeakEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-in:
			if !ok {
				return
			}
			thresh := a.Threshold()
			if snap.TotalUsage >= thresh {
				event := a.buildEvent(snap, thresh)
				if a.isDuplicate(event) {
					continue
				}
				a.mu.Lock()
				a.lastEvent = &event
				a.mu.Unlock()
				select {
				case out <- event:
				default:
				}
			}
		}
	}
}

func (a *Analyzer) buildEvent(snap model.CPUSnapshot, threshold float64) model.PeakEvent {
	return model.PeakEvent{
		ID:        generateID(snap),
		Timestamp: snap.Timestamp,
		TotalCPU:  snap.TotalUsage,
		Threshold: threshold,
		TopProcs:  snap.Processes,
	}
}

func (a *Analyzer) isDuplicate(event model.PeakEvent) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.lastEvent == nil {
		return false
	}
	if event.Timestamp.Sub(a.lastEvent.Timestamp) > a.dedupeWindow {
		return false
	}
	return fingerprint(event.TopProcs) == fingerprint(a.lastEvent.TopProcs)
}

func fingerprint(procs []model.ProcessInfo) string {
	names := make([]string, len(procs))
	for i, p := range procs {
		names[i] = p.Name
	}
	sort.Strings(names)
	h := sha256.Sum256([]byte(strings.Join(names, "|")))
	return fmt.Sprintf("%x", h[:8])
}

func generateID(snap model.CPUSnapshot) string {
	data := fmt.Sprintf("%d-%.2f", snap.Timestamp.UnixNano(), snap.TotalUsage)
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h[:8])
}
