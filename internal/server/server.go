package server

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/kdVincler/gum/internal/domain"
	"github.com/kdVincler/gum/internal/storage"
)

// Holds the shared HTTP server instance and its dependencies
type Server struct {
	store    *storage.Store
	monitors []domain.Monitor
	srv      *http.Server
}

// The JSON shape sent to clients. Struct tags are used to specify field names to the JSON encoder.
type MonitorStatus struct {
	Name      string  `json:"name"`
	URL       string  `json:"url"`
	IsUp      bool    `json:"is_up"`
	Status    int     `json:"status"`
	LatencyMs int64   `json:"latency_ms"`
	Uptime    float64 `json:"uptime_percent"`
	Error     string  `json:"error,omitempty"` // omit the error field from the JSON if it is empty
}

// Constructor for the server that creates the server with routes registered
func New(store *storage.Store, monitors []domain.Monitor, addr string) *Server {
	s := &Server{
		store:    store,
		monitors: monitors,
	}
	mux := http.NewServeMux() // Use Go's standard HTTP request multiplexer (a.k.a. router or request handler)
	mux.HandleFunc("/", s.handleStatus)
	mux.HandleFunc("/api/status", s.handleAPI)
	s.srv = &http.Server{Addr: addr, Handler: mux}
	return s
}

// Begin listening for HTTP requests
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Gracefully stop the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Give in-flight requests 5 seconds to complete before initiating a shutdown.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

// Data helper that builds the current status for all monitors
func (s *Server) getStatuses() []MonitorStatus {
	statuses := make([]MonitorStatus, 0, len(s.monitors))
	for _, m := range s.monitors {
		ms := MonitorStatus{Name: m.Name, URL: m.URL}
		if result, err := s.store.GetLatestResult(m.ID); err == nil && result != nil {
			ms.IsUp = result.IsUp
			ms.Status = result.Status
			ms.LatencyMs = result.Latency.Milliseconds()
			ms.Error = result.Error
		}
		if uptime, err := s.store.GetUptimePercent(m.ID); err == nil {
			ms.Uptime = uptime
		}
		statuses = append(statuses, ms)
	}
	return statuses
}

// JSON handler that returns the current status for all monitors
func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// stream the data received from the helper as a JSON response directly to the client through the ResponseWriter
	json.NewEncoder(w).Encode(s.getStatuses())
}

// Render the HTML status page showing the current status for all monitors.
// Uses Go's html/template package to safely render HTML with dynamic data.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("status").Parse(statusHTML))
	tmpl.Execute(w, s.getStatuses())
}

const statusHTML = `<!DOCTYPE html>
<html>
	<head>
		<title>GUM - Status</title>
		<style>
			body {
				font-family: system-ui;
				max-width: 800px;
				margin: 40px auto;
				padding: 0 20px;
				background: #F5F0E6;
				color: #2F261C;
			}
			h1 {
				color: #766044;
			}
			.monitor {
				padding: 16px;
				margin: 12px 0;
				border-radius: 8px;
				background: #E8DDC7;
			}
			.up {
				border-left: 4px solid #0cca4a;
			}
			.down {
				border-left: 4px solid #e63946;
			}
			.name {
				font-weight: bold;
				font-size: 1.1em
			}
			.meta {
				color: #5B5246;
				font-size: 0.9em;
				margin-top: 4px;
			}
			.badge {
				display: inline-block;
				padding: 2px 8px;
				border-radius: 4px;
				font-size: 0.8em;
				font-weight: bold;
			}
			.badge-up{
				background: #0cca4a;
				color: #000;
			}
			.badge-down{
				background: #e63946;
				color: #fff;
			}
		</style>
	</head>
	<body>
		<h1>GUM Status</h1>
		{{range .}}
			<div class="monitor {{if .IsUp}}up{{else}}down{{end}}">
				<div class="name">
					{{.Name}} 
					<span class="badge {{if .IsUp}}badge-up{{else}}badge-down{{end}}">
						{{if .IsUp}}UP{{else}}DOWN{{end}}
					</span>
				</div>
				<div class="meta">
					{{.URL}} | {{.LatencyMs}}ms | {{printf "%.1f" .Uptime}}% uptime
				</div>
			</div>
		{{end}}
	</body>
</html>`
