// Package memtrack provides memory tracking and profiling utilities.
package memtrack

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// MemoryTracker tracks memory usage over time.
type MemoryTracker struct {
	mu         sync.RWMutex
	samples    []MemorySample
	maxSamples int
	log        *slog.Logger
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// MemorySample represents a single memory measurement.
type MemorySample struct {
	Timestamp      time.Time `json:"timestamp"`
	RSSBytes       uint64    `json:"rss_bytes"`
	HeapAllocBytes uint64    `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64    `json:"heap_sys_bytes"`
	StackSysBytes  uint64    `json:"stack_sys_bytes"`
	Goroutines     int       `json:"goroutines"`
	GCCount        uint32    `json:"gc_count"`
	PauseTotalNs   uint64    `json:"pause_total_ns"`
}

// NewMemoryTracker creates a new memory tracker.
func NewMemoryTracker(log *slog.Logger, maxSamples int) *MemoryTracker {
	if maxSamples <= 0 {
		maxSamples = 1000
	}
	return &MemoryTracker{
		samples:    make([]MemorySample, 0, maxSamples),
		maxSamples: maxSamples,
		log:        log,
		stopCh:     make(chan struct{}),
	}
}

// Start begins periodic memory sampling.
func (mt *MemoryTracker) Start(ctx context.Context, interval time.Duration) {
	mt.wg.Add(1)
	go func() {
		defer mt.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Take initial sample
		mt.sample()

		for {
			select {
			case <-ticker.C:
				mt.sample()
			case <-mt.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the memory tracker.
func (mt *MemoryTracker) Stop() {
	close(mt.stopCh)
	mt.wg.Wait()
}

// sample takes a memory sample.
func (mt *MemoryTracker) sample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sample := MemorySample{
		Timestamp:      time.Now().UTC(),
		RSSBytes:       getRSS(),
		HeapAllocBytes: m.HeapAlloc,
		HeapSysBytes:   m.HeapSys,
		StackSysBytes:  m.StackSys,
		Goroutines:     runtime.NumGoroutine(),
		GCCount:        m.NumGC,
		PauseTotalNs:   m.PauseTotalNs,
	}

	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.samples = append(mt.samples, sample)
	if len(mt.samples) > mt.maxSamples {
		mt.samples = mt.samples[1:]
	}
}

// GetSamples returns all collected samples.
func (mt *MemoryTracker) GetSamples() []MemorySample {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	result := make([]MemorySample, len(mt.samples))
	copy(result, mt.samples)
	return result
}

// GetLatest returns the most recent sample.
func (mt *MemoryTracker) GetLatest() *MemorySample {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if len(mt.samples) == 0 {
		return nil
	}
	sample := mt.samples[len(mt.samples)-1]
	return &sample
}

// GetStats returns memory statistics.
func (mt *MemoryTracker) GetStats() MemoryStats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if len(mt.samples) == 0 {
		return MemoryStats{}
	}

	var maxRSS, maxHeapAlloc uint64
	var totalRSS, totalHeapAlloc uint64
	for _, s := range mt.samples {
		if s.RSSBytes > maxRSS {
			maxRSS = s.RSSBytes
		}
		if s.HeapAllocBytes > maxHeapAlloc {
			maxHeapAlloc = s.HeapAllocBytes
		}
		totalRSS += s.RSSBytes
		totalHeapAlloc += s.HeapAllocBytes
	}

	count := uint64(len(mt.samples))
	return MemoryStats{
		SampleCount:       count,
		MaxRSSBytes:       maxRSS,
		MaxHeapAllocBytes: maxHeapAlloc,
		AvgRSSBytes:       totalRSS / count,
		AvgHeapAllocBytes: totalHeapAlloc / count,
		CurrentRSSBytes:   mt.samples[count-1].RSSBytes,
		CurrentHeapBytes:  mt.samples[count-1].HeapAllocBytes,
		CurrentGoroutines: mt.samples[count-1].Goroutines,
		GCCount:           mt.samples[count-1].GCCount,
	}
}

// MemoryStats holds aggregated memory statistics.
type MemoryStats struct {
	SampleCount       uint64 `json:"sample_count"`
	MaxRSSBytes       uint64 `json:"max_rss_bytes"`
	MaxHeapAllocBytes uint64 `json:"max_heap_alloc_bytes"`
	AvgRSSBytes       uint64 `json:"avg_rss_bytes"`
	AvgHeapAllocBytes uint64 `json:"avg_heap_alloc_bytes"`
	CurrentRSSBytes   uint64 `json:"current_rss_bytes"`
	CurrentHeapBytes  uint64 `json:"current_heap_bytes"`
	CurrentGoroutines int    `json:"current_goroutines"`
	GCCount           uint32 `json:"gc_count"`
}

// CheckMemoryBudget checks if memory is within budget.
func (mt *MemoryTracker) CheckMemoryBudget(normalBudget, heavyBudget, ceilingBudget uint64) (string, bool) {
	stats := mt.GetStats()

	if stats.CurrentRSSBytes > ceilingBudget {
		return "critical", false
	}
	if stats.CurrentRSSBytes > heavyBudget {
		return "heavy", false
	}
	if stats.CurrentRSSBytes > normalBudget {
		return "elevated", true
	}
	return "normal", true
}

// getRSS returns the process RSS in bytes.
// This is a platform-specific implementation.
func getRSS() uint64 {
	// On Linux, we can read from /proc/self/statm or /proc/self/status
	// For cross-platform, we use a heuristic based on Go's MemStats
	// Note: This is an approximation. For accurate RSS, use platform-specific code.
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// HeapSys is the closest approximation to RSS from Go's runtime
	// In production, you'd want to use platform-specific methods
	return m.HeapSys
}

// ForceGC forces a garbage collection and returns stats before/after.
func (mt *MemoryTracker) ForceGC() (before, after MemorySample) {
	before = MemorySample{}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	before = MemorySample{
		Timestamp:      time.Now().UTC(),
		RSSBytes:       getRSS(),
		HeapAllocBytes: m.HeapAlloc,
		HeapSysBytes:   m.HeapSys,
		StackSysBytes:  m.StackSys,
		Goroutines:     runtime.NumGoroutine(),
		GCCount:        m.NumGC,
	}

	runtime.GC()
	debug.FreeOSMemory()

	runtime.ReadMemStats(&m)
	after = MemorySample{
		Timestamp:      time.Now().UTC(),
		RSSBytes:       getRSS(),
		HeapAllocBytes: m.HeapAlloc,
		HeapSysBytes:   m.HeapSys,
		StackSysBytes:  m.StackSys,
		Goroutines:     runtime.NumGoroutine(),
		GCCount:        m.NumGC,
	}

	return before, after
}

// MemoryBudget holds the memory budget configuration.
type MemoryBudget struct {
	NormalBudget  uint64 `json:"normal_budget"`  // 250 MiB
	HeavyBudget   uint64 `json:"heavy_budget"`   // 400 MiB
	CeilingBudget uint64 `json:"ceiling_budget"` // 500 MiB
}

// DefaultMemoryBudget returns the default memory budget.
func DefaultMemoryBudget() MemoryBudget {
	return MemoryBudget{
		NormalBudget:  250 * 1024 * 1024, // 250 MiB
		HeavyBudget:   400 * 1024 * 1024, // 400 MiB
		CeilingBudget: 500 * 1024 * 1024, // 500 MiB
	}
}

// String returns a human-readable representation of the memory budget status.
func (ms MemoryStats) String(budget MemoryBudget) string {
	status := "normal"
	if ms.CurrentRSSBytes > budget.CeilingBudget {
		status = "critical"
	} else if ms.CurrentRSSBytes > budget.HeavyBudget {
		status = "heavy"
	} else if ms.CurrentRSSBytes > budget.NormalBudget {
		status = "elevated"
	}

	return fmt.Sprintf("Memory: %s (RSS: %.1f MiB, Heap: %.1f MiB, Goroutines: %d, GC: %d)",
		status,
		float64(ms.CurrentRSSBytes)/1024/1024,
		float64(ms.CurrentHeapBytes)/1024/1024,
		ms.CurrentGoroutines,
		ms.GCCount)
}
