package scheduler

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kdVincler/gum/internal/domain"
	"github.com/kdVincler/gum/internal/storage"
)

// Protected map of eacah montor's call count, used for verification
type recordingChecker struct {
	mu    sync.Mutex
	calls map[string]int
}

// Mock check function for the scheduler to use. Safely records call counts too.
func (c *recordingChecker) Check(_ context.Context, monitor domain.Monitor) domain.CheckResult {
	c.mu.Lock()
	c.calls[monitor.ID]++
	c.mu.Unlock()

	return domain.CheckResult{
		MonitorID: monitor.ID,
		Status:    200,
		Latency:   25 * time.Millisecond,
		CheckedAt: time.Now(),
		IsUp:      true,
	}
}

// Safely reads the call count for a monitor
func (c *recordingChecker) callCount(monitorID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[monitorID]
}

// Temporary database for the use of these tests
func newTestStore(t *testing.T) *storage.Store {
	t.Helper()

	store, err := storage.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	return store
}

func TestRunChecksAndSavesEveryMonitorEveryRun(t *testing.T) {
	monitors := []domain.Monitor{
		{ID: "one", Name: "One", URL: "one.example"},
		{ID: "two", Name: "Two", URL: "two.example"},
		{ID: "three", Name: "Three", URL: "three.example"},
	}
	checker := &recordingChecker{calls: make(map[string]int)}
	store := newTestStore(t)
	scheduler := New(monitors, checker, store, 2)

	// runs check for each monitor twice
	for i := 1; i < 2; i++ {
		scheduler.run(context.Background())

		for _, monitor := range monitors {
			if got := checker.callCount(monitor.ID); got != i {
				t.Errorf("check count for %q = %d, want %d", monitor.ID, got, i)
			}

			result, err := store.GetLatestResult(monitor.ID)
			if err != nil {
				t.Fatalf("get result for %q: %v", monitor.ID, err)
			}
			if result.MonitorID != monitor.ID || result.Status != 200 || !result.IsUp {
				t.Errorf("saved result for %q = %+v, want successful result", monitor.ID, result)
			}
		}
	}

}

func TestRunSkipsChecksWhenContextIsCanceled(t *testing.T) {
	monitors := []domain.Monitor{
		{ID: "one", Name: "One", URL: "one.example"},
		{ID: "two", Name: "Two", URL: "two.example"},
	}
	checker := &recordingChecker{calls: make(map[string]int)}
	scheduler := New(monitors, checker, newTestStore(t), 2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scheduler.run(ctx)

	for _, monitor := range monitors {
		if got := checker.callCount(monitor.ID); got != 0 {
			t.Errorf("check count for %q = %d, want 0", monitor.ID, got)
		}
	}
}
