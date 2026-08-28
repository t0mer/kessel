// Package store is the SQLite persistence layer for Kessel.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps the application database.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

// Open opens (creating if needed) the SQLite database at path and runs
// all pending migrations.
func Open(path string) (*Store, error) {
	dsn := "file:" + url.PathEscape(path) +
		"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}
	s := &Store{db: db, now: time.Now}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the underlying *sql.DB (tests, low-level access).
func (s *Store) DB() *sql.DB { return s.db }
