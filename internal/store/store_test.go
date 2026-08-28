package store

import (
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	p := filepath.Join(t.TempDir(), "kessel.db")
	s, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenCreatesTables(t *testing.T) {
	s := openTemp(t)
	want := []string{"sites", "schedules", "runs", "threshold_rules", "channels", "notifications_log", "settings", "schema_migrations"}
	for _, tbl := range want {
		var name string
		err := s.DB().QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", tbl, err)
		}
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "kessel.db")
	s1, err := Open(p)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = s1.Close()
	s2, err := Open(p) // re-open: migrations already applied
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()
	var n int
	if err := s2.DB().QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one applied migration")
	}
}
