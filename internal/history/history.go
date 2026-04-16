package history

import (
	"sort"

	"github.com/ekorunov/watchctl/internal/model"
)

// Analyze produces a summary from a set of historical peak events.
func Analyze(events []model.PeakEvent, currentProcs []string) model.HistorySummary {
	summary := model.HistorySummary{
		TotalEvents: len(events),
	}

	if len(events) == 0 {
		return summary
	}

	// Count process name frequency across all events.
	freq := make(map[string]int)
	for _, ev := range events {
		seen := make(map[string]bool)
		for _, p := range ev.TopProcs {
			if !seen[p.Name] {
				freq[p.Name]++
				seen[p.Name] = true
			}
		}
	}

	// Build sorted frequency list.
	for name, count := range freq {
		if count > 1 {
			summary.FrequentProcs = append(summary.FrequentProcs, model.ProcFrequency{
				Name:  name,
				Count: count,
			})
		}
	}
	sort.Slice(summary.FrequentProcs, func(i, j int) bool {
		return summary.FrequentProcs[i].Count > summary.FrequentProcs[j].Count
	})

	// Find new processes: appear in current snapshot but not in history.
	historicalNames := make(map[string]bool)
	for name := range freq {
		historicalNames[name] = true
	}
	for _, name := range currentProcs {
		if !historicalNames[name] {
			summary.NewProcs = append(summary.NewProcs, name)
		}
	}

	// Recent events — last 20.
	n := 20
	if n > len(events) {
		n = len(events)
	}
	summary.RecentEvents = events[len(events)-n:]

	return summary
}
