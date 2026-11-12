# GUM - Go Uptime Monitor

GUM is a lightweight uptime monitor for HTTP and TCP services. It concurrently checks configured endpoints at set intervals, stores the results in SQLite, and serves a simple status page and JSON API so you can quickly see whether a service is up or down.

I wanted to learn Go, and I tend to learn a new programming language by building something, so I made this.

## Usage

Run the monitor from the project root:

```bash
go run ./cmd/gum
```

The app starts a local status page at `http://localhost:8080` and begins monitoring the built-in endpoints. It also writes results to a SQLite database file named `gum.db` in the project root.

To build/update a binary instead of running directly:

```bash
go build -o gum ./cmd/gum
```

Then run the built binary by:

```bash
./gum
```

Use `Ctrl+C` to stop the service gracefully.

### What the app checks

By default, GUM monitors a small set of endpoints including:

- HTTP checks to `https://httpbin.org/status/200`
- HTTP checks to `https://httpbin.org/status/500`
- HTTP checks to `https://go.dev`
- A TCP check to `google.com:443`

You can change the monitored endpoints by updating the monitor slices in `cmd/gum/main.go`. Note that the scheduler worker counts are currently hardcoded to match the number of monitors for each check type.

The scheduler runs each monitor at regular intervals (30 seconds) and prints a short status line to the terminal while saving the result to the database.

## How it works

GUM is built around a simple pattern:

- `internal/checker` performs the actual health checks for HTTP and TCP endpoints.
- `internal/scheduler` runs those checks concurrently with a configurable worker pool.
- `internal/storage` persists each result in SQLite so the project can calculate uptime percentages.
- `internal/server` exposes the status page at `/` and the machine-readable data at `/api/status`.

The status page renders a list-like HTML view showing each monitor's name, URL, response time, uptime, and current health. The API endpoint returns the same information as JSON for integrations or dashboards.

## Project structure

- `cmd/gum/main.go` — starts the schedulers, database, and HTTP server.
- `internal/checker/` — HTTP and TCP checker implementations.
- `internal/scheduler/` — concurrent monitoring loop.
- `internal/server/` — status page and JSON API.
- `internal/storage/` — SQLite persistence layer.
- `internal/domain/` — shared monitor and check result models.
- `gum.db` — SQLite database created automatically when the app runs.

## Testing

The project includes focused unit tests for the internal logic of the module.

Run the suite with:

```bash
go test ./...
```
