// This package holds the modules data types without any external dependencies.
package domain

import "time"

// A single monitored URL endpoint and its metadata
type Monitor struct {
	ID   string
	Name string
	URL  string
}

// The result of one probe attempt of a monitored endpoint
type CheckResult struct {
	MonitorID string
	Status    int
	Latency   time.Duration
	Error     string
	CheckedAt time.Time
	IsUp      bool
}
