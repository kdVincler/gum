package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kdVincler/gum/internal/domain"
)

func TestHTTPCheckerCheckValidURL(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		wantIsUp bool
	}{
		{name: "successful response", status: http.StatusOK, wantIsUp: true},
		{name: "redirect response", status: http.StatusFound, wantIsUp: true},
		{name: "not found error", status: http.StatusNotFound, wantIsUp: false},
		{name: "server error", status: http.StatusInternalServerError, wantIsUp: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup test webserver to act as a webpage to check
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status == http.StatusFound {
					w.Header().Set("Location", "/redirect-target")
				}
				w.WriteHeader(tt.status)
			}))
			defer ts.Close()

			checker := NewHTTPChecker(time.Second)
			// configure the test http client so it doesn't follow the redirects
			checker.client.CheckRedirect = func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			}

			result := checker.Check(context.Background(), domain.Monitor{
				ID:  "monitor-1",
				URL: ts.URL,
			})

			if result.MonitorID != "monitor-1" {
				t.Errorf("MonitorID = %q, want %q", result.MonitorID, "monitor-1")
			}
			if result.Status != tt.status {
				t.Errorf("Status = %d, want %d", result.Status, tt.status)
			}
			if result.IsUp != tt.wantIsUp {
				t.Errorf("IsUp = %t, want %t", result.IsUp, tt.wantIsUp)
			}
			if result.Error != "" {
				t.Errorf("Error = %q, want empty", result.Error)
			}
			if result.CheckedAt.IsZero() {
				t.Error("CheckedAt is zero")
			}
			if result.Latency < 0 {
				t.Errorf("Latency = %s, want non-negative", result.Latency)
			}
		})
	}
}

func TestHTTPCheckerCheckInvalidURL(t *testing.T) {
	result := NewHTTPChecker(time.Second).Check(context.Background(), domain.Monitor{
		ID:  "monitor-invalid",
		URL: "://invalid",
	})

	if result.MonitorID != "monitor-invalid" {
		t.Errorf("MonitorID = %q, want %q", result.MonitorID, "monitor-invalid")
	}
	if result.Status != 0 {
		t.Errorf("Status = %d, want 0", result.Status)
	}
	if result.IsUp {
		t.Error("IsUp = true, want false")
	}
	if result.Error == "" {
		t.Error("Error is empty, want request construction error")
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt is zero")
	}
}
