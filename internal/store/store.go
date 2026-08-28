// Package store is the SQLite persistence layer for Kessel.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps the application database. The underlying *sql.DB is guarded by a
// RWMutex so it can be swapped atomically during a restore.
type Store struct {
	mu    sync.RWMutex
	sqldb *sql.DB
	path  string
	now   func() time.Time
}

func dsnFor(path string) string {
	return "file:" + url.PathEscape(path) +
		"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

func openSQL(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsnFor(path))
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}
	return db, nil
}

// Open opens (creating if needed) the SQLite database at path and runs
// all pending migrations.
func Open(path string) (*Store, error) {
	db, err := openSQL(path)
	if err != nil {
		return nil, err
	}
	s := &Store{sqldb: db, path: path, now: time.Now}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return s, nil
}

// conn returns the current database handle. Callers use the returned handle
// immediately; a concurrent restore may close it, surfacing as a query error
// rather than a data race.
func (s *Store) conn() *sql.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sqldb
}

// Close closes the database.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sqldb.Close()
}

// DB exposes the underlying *sql.DB (tests, low-level access).
func (s *Store) DB() *sql.DB { return s.conn() }
