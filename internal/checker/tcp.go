package checker

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/kdVincler/gum/internal/domain"
)

/*

This Checker interface uses TCP dials on raw ports to check for the uptime services
not reachable through HTTP (like databases, redis instances, SMTP servers etc.).

*/

// Shared checker holds timeout duration for TCP dial attempts
type TCPChecker struct {
	timeout time.Duration
}

// Constructor for the checker
func NewTCPChecker(timeout time.Duration) *TCPChecker {
	return &TCPChecker{timeout: timeout}
}

/*
Perform a single check of a monitored endpoint and return the result.

Attempts to connect to an address (URL:PORT).
If connection times out or is refused, marks it as DOWN, otherwise marks it as UP

CheckResult.Status will always be 0 (TCP doesn't have status codes)
*/
func (c *TCPChecker) Check(ctx context.Context, monitor domain.Monitor) domain.CheckResult {
	// save start time for latency calculation
	s := time.Now()

	// Attempt TCP connection
	conn, err := net.DialTimeout("tcp", monitor.URL, c.timeout)
	latency := time.Since(s)

	if err != nil {
		return domain.CheckResult{
			MonitorID: monitor.ID,
			Status:    0,
			Latency:   latency,
			Error:     fmt.Sprintf("tcp dial failed: %v", err),
			CheckedAt: time.Now(),
			IsUp:      false,
		}
	}

	// Clean up (close) TCP connection
	conn.Close()

	return domain.CheckResult{
		MonitorID: monitor.ID,
		Status:    0,
		Latency:   latency,
		Error:     "",
		CheckedAt: time.Now(),
		IsUp:      true,
	}
}
