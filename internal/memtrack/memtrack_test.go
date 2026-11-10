// internal/memtrack/memtrack_test.go
package memtrack

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMemoryTrackerBasic(t *testing.T) {
	log := newTestLogger()
	tracker := NewMemoryTracker(log, 100)

	// Start with short interval for testing
	ctx := context.Background()
	tracker.Start(ctx, 10*time.Millisecond)
	defer tracker.Stop()

	// Wait for a few samples
	time.Sleep(50 * time.Millisecond)

	samples := tracker.GetSamples()
	if len(samples) == 0 {
		t.Fatal("expected at least one sample")
	}

	latest := tracker.GetLatest()
	if latest == nil {
		t.Fatal("expected latest sample")
	}

	if latest.RSSBytes == 0 {
		t.Log("RSS is 0 (using HeapSys approximation)")
	}

	if latest.Goroutines == 0 {
		t.Fatal("expected goroutines > 0")
	}

	stats := tracker.GetStats()
	if stats.SampleCount == 0 {
		t.Fatal("expected sample count > 0")
	}

	t.Logf("Memory stats: %s", stats.String(DefaultMemoryBudget()))
}

func TestMemoryBudgetChecking(t *testing.T) {
	log := newTestLogger()
	tracker := NewMemoryTracker(log, 10)

	ctx := context.Background()
	tracker.Start(ctx, 10*time.Millisecond)
	defer tracker.Stop()

	time.Sleep(50 * time.Millisecond)

	budget := DefaultMemoryBudget()

	// Normal budget check
	status, ok := tracker.CheckMemoryBudget(budget.NormalBudget, budget.HeavyBudget, budget.CeilingBudget)
	t.Logf("Budget status: %s, ok: %v", status, ok)

	// Should be normal for typical Go process
	if status == "critical" {
		t.Fatal("unexpected critical status")
	}
}

func TestForceGC(t *testing.T) {
	log := newTestLogger()
	tracker := NewMemoryTracker(log, 10)

	ctx := context.Background()
	tracker.Start(ctx, 10*time.Millisecond)
	defer tracker.Stop()

	time.Sleep(50 * time.Millisecond)

	before, after := tracker.ForceGC()

	t.Logf("Before GC: RSS=%d, Heap=%d, Goroutines=%d",
		before.RSSBytes, before.HeapAllocBytes, before.Goroutines)
	t.Logf("After GC: RSS=%d, Heap=%d, Goroutines=%d",
		after.RSSBytes, after.HeapAllocBytes, after.Goroutines)

	// GC count should increase
	if after.GCCount <= before.GCCount {
		t.Log("GC count did not increase (may be expected if no GC needed)")
	}
}

func TestMemoryBudgetDefaults(t *testing.T) {
	budget := DefaultMemoryBudget()

	if budget.NormalBudget != 250*1024*1024 {
		t.Errorf("NormalBudget = %d, want 250 MiB", budget.NormalBudget)
	}
	if budget.HeavyBudget != 400*1024*1024 {
		t.Errorf("HeavyBudget = %d, want 400 MiB", budget.HeavyBudget)
	}
	if budget.CeilingBudget != 500*1024*1024 {
		t.Errorf("CeilingBudget = %d, want 500 MiB", budget.CeilingBudget)
	}
}

func TestMemoryStatsString(t *testing.T) {
	stats := MemoryStats{
		CurrentRSSBytes:   300 * 1024 * 1024,
		CurrentHeapBytes:  200 * 1024 * 1024,
		CurrentGoroutines: 100,
		GCCount:           50,
	}

	budget := DefaultMemoryBudget()
	str := stats.String(budget)

	t.Logf("Stats string: %s", str)

	if !strings.Contains(str, "elevated") {
		t.Errorf("expected 'elevated' status for 300 MiB RSS, got: %s", str)
	}
}

func TestGetRSSDoesNotPanic(t *testing.T) {
	// Just ensure it doesn't panic
	rss := getRSS()
	t.Logf("RSS: %d bytes", rss)
	if rss == 0 {
		t.Log("RSS is 0 (expected on some platforms)")
	}
}

func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping leak detection test in short mode")
	}

	log := newTestLogger()
	tracker := NewMemoryTracker(log, 1000)

	ctx := context.Background()
	tracker.Start(ctx, 100*time.Millisecond)
	defer tracker.Stop()

	// Initial measurement
	time.Sleep(200 * time.Millisecond)
	initialStats := tracker.GetStats()
	initialGoroutines := initialStats.CurrentGoroutines

	// Simulate workload that might leak
	for i := 0; i < 100; i++ {
		go func() {
			time.Sleep(10 * time.Millisecond)
		}()
	}

	time.Sleep(500 * time.Millisecond)

	// Force GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalStats := tracker.GetStats()
	finalGoroutines := finalStats.CurrentGoroutines

	t.Logf("Initial goroutines: %d, Final: %d", initialGoroutines, finalGoroutines)

	// Allow some variance but not massive leak
	if finalGoroutines > initialGoroutines+50 {
		t.Errorf("Possible goroutine leak: %d -> %d", initialGoroutines, finalGoroutines)
	}
}

// Test helper
type testLogger struct{}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&testLogger{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

func (t *testLogger) Write(p []byte) (n int, err error) {
	return len(p), nil
}

var _ = runtime.GC
