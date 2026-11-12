package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdVincler/gum/internal/domain"
	"github.com/kdVincler/gum/internal/storage"
)

// Helper that sets up a temporary database and a server to use for tests
func newTestServer(t *testing.T, monitors []domain.Monitor) (*Server, *storage.Store) {
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
	return New(store, monitors, "127.0.0.1:0"), store
}

func TestHandleAPI(t *testing.T) {
	monitors := []domain.Monitor{
		{ID: "up", Name: "Healthy", URL: "https://healthy.example"},
		{ID: "down", Name: "Unhealthy", URL: "https://unhealthy.example"},
	}
	server, store := newTestServer(t, monitors)
	checkedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	// add data to the temp db
	for _, result := range []domain.CheckResult{
		{MonitorID: "up", Status: 200, Latency: 45 * time.Millisecond, CheckedAt: checkedAt, IsUp: true},
		{MonitorID: "down", Status: 503, Latency: 80 * time.Millisecond, Error: "unavailable", CheckedAt: checkedAt, IsUp: false},
	} {
		if err := store.SaveResult(result); err != nil {
			t.Fatalf("save result: %v", err)
		}
	}

	// query the test server for API response
	recorder := httptest.NewRecorder()
	server.handleAPI(recorder, httptest.NewRequest(http.MethodGet, "/api/status", nil))

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	// convert json back into MonitorStatus and check each entry
	var got []MonitorStatus
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode API response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d monitor statuses, want 2", len(got))
	}
	if got[0].Name != "Healthy" || !got[0].IsUp || got[0].Status != 200 || got[0].LatencyMs != 45 || got[0].Uptime != 100 {
		t.Errorf("first status = %+v, want healthy monitor status", got[0])
	}
	if got[1].Name != "Unhealthy" || got[1].IsUp || got[1].Status != 503 || got[1].LatencyMs != 80 || got[1].Uptime != 0 || got[1].Error != "unavailable" {
		t.Errorf("second status = %+v, want unhealthy monitor status", got[1])
	}
}

func TestHandleStatusTemplateEscapesMonitorContent(t *testing.T) {
	monitor := domain.Monitor{
		ID:   "untrusted",
		Name: `<script>alert("x")</script>`,
		URL:  `https://example.test/?q=<tag>`,
	}
	server, _ := newTestServer(t, []domain.Monitor{monitor})

	// query the server for the status page
	recorder := httptest.NewRecorder()
	server.handleStatus(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	// check the returned HTML
	body := recorder.Body.String()
	if strings.Contains(body, monitor.Name) {
		t.Error("response contains unescaped monitor name")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("response does not contain escaped monitor name")
	}
	if strings.Contains(body, "<tag>") {
		t.Error("response contains unescaped monitor URL")
	}
}
