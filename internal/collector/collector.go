package collector

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/ekorunov/watchctl/internal/model"
)

// Collector gathers CPU metrics at regular intervals.
// It caches process handles to get accurate CPU% deltas between ticks.
type Collector struct {
	interval time.Duration
	topN     int
	numCPU   float64

	mu      sync.Mutex
	handles map[int32]*process.Process
}

// New creates a Collector with the given poll interval and top-N limit.
func New(interval time.Duration, topN int) *Collector {
	return &Collector{
		interval: interval,
		topN:     topN,
		numCPU:   float64(runtime.NumCPU()),
		handles:  make(map[int32]*process.Process),
	}
}

// Run starts the collection loop, sending snapshots to out until ctx is cancelled.
func (c *Collector) Run(ctx context.Context, out chan<- model.CPUSnapshot) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := c.collect(ctx)
			select {
			case out <- snap:
			default:
			}
		}
	}
}

func (c *Collector) collect(ctx context.Context) model.CPUSnapshot {
	now := time.Now()

	percents, err := cpu.PercentWithContext(ctx, 500*time.Millisecond, false)
	totalUsage := 0.0
	if err == nil && len(percents) > 0 {
		totalUsage = percents[0]
	}

	procs := c.topProcesses(ctx)

	return model.CPUSnapshot{
		Timestamp:  now,
		TotalUsage: totalUsage,
		Processes:  procs,
	}
}

func (c *Collector) topProcesses(ctx context.Context) []model.ProcessInfo {
	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		return nil
	}

	// Build set of alive PIDs for cleanup.
	alive := make(map[int32]bool, len(pids))
	for _, pid := range pids {
		alive[pid] = true
	}

	c.mu.Lock()
	// Remove stale handles.
	for pid := range c.handles {
		if !alive[pid] {
			delete(c.handles, pid)
		}
	}
	c.mu.Unlock()

	type entry struct {
		info model.ProcessInfo
	}
	var entries []entry

	for _, pid := range pids {
		p := c.getOrCreate(pid, ctx)
		if p == nil {
			continue
		}

		cpuPct, err := p.CPUPercentWithContext(ctx)
		if err != nil {
			continue
		}
		// Normalize: gopsutil returns per-core %, divide by NumCPU to match Task Manager.
		cpuPct = cpuPct / c.numCPU

		name, _ := p.NameWithContext(ctx)
		createTime, _ := p.CreateTimeWithContext(ctx)
		exe, _ := p.ExeWithContext(ctx)

		entries = append(entries, entry{
			info: model.ProcessInfo{
				PID:        pid,
				Name:       name,
				Exe:        exe,
				CPUPercent: cpuPct,
				CreateTime: createTime,
			},
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].info.CPUPercent > entries[j].info.CPUPercent
	})

	n := c.topN
	if n > len(entries) {
		n = len(entries)
	}

	result := make([]model.ProcessInfo, n)
	for i := 0; i < n; i++ {
		result[i] = entries[i].info
	}
	return result
}

func (c *Collector) getOrCreate(pid int32, ctx context.Context) *process.Process {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p, ok := c.handles[pid]; ok {
		return p
	}

	p, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil
	}
	c.handles[pid] = p
	return p
}
