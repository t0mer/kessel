package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupCreatesValidSnapshot(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, Site{Name: "One", URL: "https://one.com", Strategy: StrategyMobile})

	dest := filepath.Join(t.TempDir(), "backup.db")
	if err := s.Backup(ctx, dest); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if fi, err := os.Stat(dest); err != nil || fi.Size() == 0 {
		t.Fatalf("backup file missing or empty: %v", err)
	}

	// The snapshot is itself a usable Kessel DB with the data.
	restored, err := Open(dest)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer restored.Close()
	sites, _ := restored.ListSites(ctx)
	if len(sites) != 1 || sites[0].Name != "One" {
		t.Fatalf("backup contents wrong: %+v", sites)
	}
}

func TestRestoreFromReplacesData(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, Site{Name: "One", URL: "https://one.com", Strategy: StrategyMobile})

	// Snapshot with just "One".
	backup := filepath.Join(t.TempDir(), "snap.db")
	if err := s.Backup(ctx, backup); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	// Mutate the live DB after the snapshot.
	_, _ = s.CreateSite(ctx, Site{Name: "Two", URL: "https://two.com", Strategy: StrategyMobile})
	if sites, _ := s.ListSites(ctx); len(sites) != 2 {
		t.Fatalf("expected 2 sites before restore, got %d", len(sites))
	}

	// Restore rolls back to the snapshot state.
	if err := s.RestoreFrom(ctx, backup); err != nil {
		t.Fatalf("RestoreFrom: %v", err)
	}
	sites, err := s.ListSites(ctx)
	if err != nil {
		t.Fatalf("ListSites after restore: %v", err)
	}
	if len(sites) != 1 || sites[0].Name != "One" {
		t.Fatalf("after restore = %+v, want only One", sites)
	}
	// Store keeps working (new writes ok).
	if _, err := s.CreateSite(ctx, Site{Name: "Three", URL: "https://three.com", Strategy: StrategyMobile}); err != nil {
		t.Fatalf("write after restore: %v", err)
	}
}

func TestRestoreRejectsInvalidFile(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, Site{Name: "One", URL: "https://one.com", Strategy: StrategyMobile})

	bad := filepath.Join(t.TempDir(), "notadb.txt")
	if err := os.WriteFile(bad, []byte("this is not a sqlite database"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreFrom(ctx, bad); err == nil {
		t.Fatal("expected restore to reject a non-SQLite file")
	}
	// Original data intact and store still usable.
	sites, err := s.ListSites(ctx)
	if err != nil {
		t.Fatalf("ListSites after failed restore: %v", err)
	}
	if len(sites) != 1 || sites[0].Name != "One" {
		t.Fatalf("data lost after failed restore: %+v", sites)
	}
}
