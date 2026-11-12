package storage

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite" // Import the Go native SQLite driver for database/sql to use

	"github.com/kdVincler/gum/internal/domain"
)

// Shared database that wraps the standard sql.DB connection
type Store struct {
	db *sql.DB
}

// Constructor for the Store that opens or creates a SQLite database file connection and returns a Store instance.
func New(path string) (*Store, error) {
	// Open a SQLite database connection with Write Ahead Logging (WAL) mode
	// (so status page can read while the scheduler writes) and a busy timeout of 5 seconds (waits up to 5 seconds if the database is locked).
	// If the passed database file does not exist, it will be created, courtesy of the SQLite driver. If it does exist, it will be opened.
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}

	// Ensure the check_results table exists in the opened database, creating it if it does not
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS check_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			monitor_id TEXT NOT NULL,
			status INTEGER NOT NULL,
			latency_ms INTEGER NOT NULL,
			error TEXT,
			is_up BOOLEAN NOT NULL,
			checked_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close the opened database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Insert passed CheckResult values into the check_results table in the database. Returns an error if the insert fails.
func (s *Store) SaveResult(r domain.CheckResult) error {
	_, err := s.db.Exec(
		"INSERT INTO check_results (monitor_id, status, latency_ms, error, is_up, checked_at) VALUES (?, ?, ?, ?, ?, ?)",
		r.MonitorID,
		r.Status,
		r.Latency.Milliseconds(), // Store latency in milliseconds as an integer instead of a duration string
		r.Error,
		r.IsUp,
		r.CheckedAt,
	)
	return err
}

// Calculate the uptime percentage for a given monitor
func (s *Store) GetUptimePercent(monitorID string) (float64, error) {
	var total, up int
	// Query the database for the total number of checks and the number of successful checks for the given monitor ID
	// Use COALESCE to handle the case where there are no successful checks (SUM would return NULL, COALESCE converts it to 0)
	err := s.db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_up THEN 1 ELSE 0 END), 0) FROM check_results WHERE monitor_id = ?",
		monitorID,
	).Scan(&total, &up)
	if err != nil || total == 0 {
		// if the query fails or there are no checks for that monitor, return 100% uptime and the error (if any)
		return 100.0, err
	}
	return float64(up) / float64(total) * 100.0, nil
}

// Retrieve the latest check result for a given monitor ID from the database.
func (s *Store) GetLatestResult(monitorID string) (*domain.CheckResult, error) {
	// Query the database for all columns of the single most recent result for the given monitor ID.
	row := s.db.QueryRow(
		"SELECT monitor_id, status, latency_ms, error, is_up, checked_at FROM check_results WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT 1",
		monitorID,
	)

	var r domain.CheckResult
	var latencyMs int64
	var errStr sql.NullString // Use sql.NullString to handle the case where the error column is NULL in the database (.Scan would fail)
	err := row.Scan(&r.MonitorID, &r.Status, &latencyMs, &errStr, &r.IsUp, &r.CheckedAt)
	if err != nil {
		return nil, err
	}
	r.Latency = time.Duration(latencyMs) * time.Millisecond // Convert latency from int milliseconds back to a time.Duration
	if errStr.Valid {
		r.Error = errStr.String // only set the error of the return value if it is not NULL in the database
	}
	return &r, nil
}
