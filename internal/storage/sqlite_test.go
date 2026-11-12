package storage

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdVincler/gum/internal/domain"
)

// Temporary database for the use of these tests
func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := New(filepath.Join(t.TempDir(), "test.db"))
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

func TestStoreSaveAndGetLatestResult(t *testing.T) {
	store := newTestStore(t)
	older := domain.CheckResult{
		MonitorID: "monitor-1",
		Status:    200,
		Latency:   1250 * time.Millisecond,
		CheckedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		IsUp:      true,
	}
	newer := domain.CheckResult{
		MonitorID: "monitor-1",
		Status:    503,
		Latency:   2250 * time.Millisecond,
		Error:     "service unavailable",
		CheckedAt: older.CheckedAt.Add(time.Minute),
		IsUp:      false,
	}

	// Populate temp DB
	for _, result := range []domain.CheckResult{older, newer} {
		if err := store.SaveResult(result); err != nil {
			t.Fatalf("save result: %v", err)
		}
	}

	got, err := store.GetLatestResult("monitor-1")
	if err != nil {
		t.Fatalf("get latest result: %v", err)
	}
	if got.MonitorID != newer.MonitorID || got.Status != newer.Status {
		t.Errorf("latest result = %+v, want monitor %q with status %d", got, newer.MonitorID, newer.Status)
	}
	if got.Latency != newer.Latency {
		t.Errorf("Latency = %s, want %s", got.Latency, newer.Latency)
	}
	if got.Error != newer.Error {
		t.Errorf("Error = %q, want %q", got.Error, newer.Error)
	}
	if got.IsUp != newer.IsUp {
		t.Errorf("IsUp = %t, want %t", got.IsUp, newer.IsUp)
	}
	if !got.CheckedAt.Equal(newer.CheckedAt) {
		t.Errorf("CheckedAt = %s, want %s", got.CheckedAt, newer.CheckedAt)
	}
}

func TestStoreGetLatestResultNoResults(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetLatestResult("missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("error = %v, want %v", err, sql.ErrNoRows)
	}
}

func TestStoreGetUptimePercent(t *testing.T) {
	store := newTestStore(t)
	checkedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		monitorID string
		isUp      bool
	}{
		{name: "first up check", monitorID: "mixed", isUp: true},
		{name: "second down check", monitorID: "mixed", isUp: false},
		{name: "only up check", monitorID: "all-up", isUp: true},
	}

	// Populate test DB
	for i, tt := range tests {
		if err := store.SaveResult(domain.CheckResult{
			MonitorID: tt.monitorID,
			IsUp:      tt.isUp,
			CheckedAt: checkedAt.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("save %s: %v", tt.name, err)
		}
	}

	for _, tt := range []struct {
		monitorID string
		want      float64
	}{
		{monitorID: "mixed", want: 50},
		{monitorID: "all-up", want: 100},
		{monitorID: "no-checks", want: 100},
	} {
		got, err := store.GetUptimePercent(tt.monitorID)
		if err != nil {
			t.Fatalf("get uptime for %q: %v", tt.monitorID, err)
		}
		if got != tt.want {
			t.Errorf("uptime for %q = %v, want %v", tt.monitorID, got, tt.want)
		}
	}
}
