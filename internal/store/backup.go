package store

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Backup writes a consistent snapshot of the live database to destPath using
// SQLite's VACUUM INTO (no downtime, no locking of the app). destPath must not
// already exist and its parent directory must exist.
func (s *Store) Backup(ctx context.Context, destPath string) error {
	if _, err := s.conn().ExecContext(ctx, "VACUUM INTO ?", destPath); err != nil {
		return fmt.Errorf("vacuum into %s: %w", destPath, err)
	}
	return nil
}

// RestoreFrom replaces the live database with the SQLite file at srcPath.
// The file is validated first; the current DB is moved aside so a failure can
// roll back. Callers should reload dependent state (e.g. the scheduler) after.
func (s *Store) RestoreFrom(ctx context.Context, srcPath string) error {
	if err := validateKesselDB(ctx, srcPath); err != nil {
		return fmt.Errorf("invalid backup file: %w", err)
	}

	s.mu.Lock()
	if err := s.sqldb.Close(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("closing current database: %w", err)
	}

	bak := s.path + ".prerestore"
	_ = os.Remove(bak)
	if err := os.Rename(s.path, bak); err != nil {
		// Nothing moved; try to reopen the original and bail.
		s.reopenLocked()
		s.mu.Unlock()
		return fmt.Errorf("staging current database: %w", err)
	}
	_ = os.Remove(s.path + "-wal")
	_ = os.Remove(s.path + "-shm")

	if err := copyFile(srcPath, s.path); err != nil {
		s.rollbackLocked(bak)
		s.mu.Unlock()
		return fmt.Errorf("writing restored database: %w", err)
	}

	db, err := openSQL(s.path)
	if err != nil {
		s.rollbackLocked(bak)
		s.mu.Unlock()
		return fmt.Errorf("reopening restored database: %w", err)
	}
	s.sqldb = db
	s.mu.Unlock()

	// Bring an older backup up to the current schema.
	if err := s.migrate(); err != nil {
		return fmt.Errorf("migrating restored database: %w", err)
	}
	_ = os.Remove(bak)
	return nil
}

// reopenLocked reopens s.path into s.sqldb. Caller must hold s.mu.
func (s *Store) reopenLocked() {
	if db, err := openSQL(s.path); err == nil {
		s.sqldb = db
	}
}

// rollbackLocked restores the pre-restore copy and reopens it. Caller holds s.mu.
func (s *Store) rollbackLocked(bak string) {
	_ = os.Remove(s.path)
	_ = os.Rename(bak, s.path)
	s.reopenLocked()
}

// validateKesselDB checks that path is a healthy SQLite database that looks
// like a Kessel database (has the schema_migrations table).
func validateKesselDB(ctx context.Context, path string) error {
	if fi, err := os.Stat(path); err != nil {
		return fmt.Errorf("reading file: %w", err)
	} else if fi.Size() == 0 {
		return fmt.Errorf("file is empty")
	}
	db, err := openSQL(path)
	if err != nil {
		return err
	}
	defer db.Close()

	var result string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return fmt.Errorf("integrity check: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`).Scan(&n); err != nil {
		return fmt.Errorf("reading schema: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("not a Kessel database (no schema_migrations table)")
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
