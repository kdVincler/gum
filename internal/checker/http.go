package checker

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/kdVincler/gum/internal/domain"
)

// Shared checker that wraps standard http.Client with a timeout
type HTTPChecker struct {
	client *http.Client
}

// Constructor for the checker that allows configuring per request timeout for the HTTP requests
func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Perform a single check of a monitored endpoint and return the result.
func (c *HTTPChecker) Check(ctx context.Context, monitor domain.Monitor) domain.CheckResult {
	// save start time for latency calculation
	s := time.Now()

	// Construct context aware request. Uses context to allow for graceful cancelation
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, monitor.URL, nil)
	// if request creation fails, return a failed check result
	if err != nil {
		return domain.CheckResult{
			MonitorID: monitor.ID,
			Status:    0,
			Latency:   time.Since(s),
			Error:     err.Error(),
			CheckedAt: time.Now(),
			IsUp:      false,
		}
	}

	// Perform the constructed request (and measure latency)
	resp, err := c.client.Do(req)
	latency := time.Since(s)

	// if request fails, return a failed check result
	if err != nil {
		return domain.CheckResult{
			MonitorID: monitor.ID,
			Status:    0,
			Latency:   latency,
			Error:     err.Error(),
			CheckedAt: time.Now(),
			IsUp:      false,
		}
	}

	// Drain and close the body to free the TCP connection
	// Standard Go behavior. Have to do this because we don't need the response body but
	// net/http reuses connections and if we don't drain (a.k.a. read till EOF) and close
	// the body, the conneciton will be left open and we eventually run out of available connections.
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// return the executed check result with the status code and latency
	return domain.CheckResult{
		MonitorID: monitor.ID,
		Status:    resp.StatusCode,
		Latency:   latency,
		Error:     "",
		CheckedAt: time.Now(),
		IsUp:      resp.StatusCode >= 200 && resp.StatusCode < 400, // consider 2xx and 3xx as up
	}
}
