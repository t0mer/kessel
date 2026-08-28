package store

import (
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

func (s *Store) migrate() error {
	if _, err := s.db.Exec(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version, err := versionFromName(name)
		if err != nil {
			return err
		}
		var applied int
		if err := s.db.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("checking migration %d: %w", version, err)
		}
		if applied > 0 {
			continue
		}
		sqlBytes, err := fs.ReadFile(migrationsFS, "migrations/"+name)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("applying %s: %w", name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
			version, s.now().Unix(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("recording %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}
	return nil
}

// versionFromName parses the leading integer of a migration filename,
// e.g. "0001_init.sql" -> 1.
func versionFromName(name string) (int, error) {
	base := name
	if i := strings.IndexByte(base, '_'); i >= 0 {
		base = base[:i]
	} else {
		base = strings.TrimSuffix(base, ".sql")
	}
	v, err := strconv.Atoi(base)
	if err != nil {
		return 0, fmt.Errorf("invalid migration filename %q: %w", name, err)
	}
	return v, nil
}
