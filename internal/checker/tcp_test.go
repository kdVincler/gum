package checker

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kdVincler/gum/internal/domain"
)

func TestTCPCheckerCheckValidAddress(t *testing.T) {
	// setup test TCP listener to check
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	result := NewTCPChecker(time.Second).Check(context.Background(), domain.Monitor{
		ID:  "tcp-monitor",
		URL: listener.Addr().String(),
	})

	if result.MonitorID != "tcp-monitor" {
		t.Errorf("MonitorID = %q, want %q", result.MonitorID, "tcp-monitor")
	}
	if !result.IsUp {
		t.Errorf("IsUp = false, want true; error: %s", result.Error)
	}
	if result.Status != 0 {
		t.Errorf("Status = %d, want 0 for TCP", result.Status)
	}
	if result.Error != "" {
		t.Errorf("Error = %q, want empty", result.Error)
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt is zero")
	}
}

func TestTCPCheckerCheckInvalidAddress(t *testing.T) {
	result := NewTCPChecker(time.Second).Check(context.Background(), domain.Monitor{
		ID:  "tcp-invalid",
		URL: "not-an-address",
	})

	if result.MonitorID != "tcp-invalid" {
		t.Errorf("MonitorID = %q, want %q", result.MonitorID, "tcp-invalid")
	}
	if result.IsUp {
		t.Error("IsUp = true, want false")
	}
	if result.Status != 0 {
		t.Errorf("Status = %d, want 0 for TCP", result.Status)
	}
	if result.Error == "" {
		t.Error("Error is empty, want dial error")
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt is zero")
	}
}
