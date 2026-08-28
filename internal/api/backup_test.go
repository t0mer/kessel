package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/t0mer/kessel/internal/store"
)

func TestCreateListDownloadBackup(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "One", URL: "https://one.com", Strategy: store.StrategyMobile})

	// Create
	rec := do(t, a, http.MethodPost, "/backups", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create -> %d (body=%s)", rec.Code, rec.Body.String())
	}
	var info backupInfo
	decode(t, rec, &info)
	if info.Name == "" || info.Size == 0 {
		t.Fatalf("unexpected backup info: %+v", info)
	}

	// List
	rec = do(t, a, http.MethodGet, "/backups", "")
	var list []backupInfo
	decode(t, rec, &list)
	if len(list) != 1 || list[0].Name != info.Name {
		t.Fatalf("list = %+v, want the created backup", list)
	}

	// Download (attachment)
	rec = do(t, a, http.MethodGet, "/backups/"+info.Name, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("download -> %d", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd == "" {
		t.Error("download missing Content-Disposition attachment header")
	}
	if rec.Body.Len() == 0 {
		t.Error("download body empty")
	}
}

func TestDownloadBackupTraversalBlocked(t *testing.T) {
	a, _, _, _ := newAPI(t)
	rec := do(t, a, http.MethodGet, "/backups/..%2f..%2fetc%2fpasswd", "")
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Fatalf("traversal -> %d, want 400/404", rec.Code)
	}
}

func TestRestoreFromExistingBackup(t *testing.T) {
	a, s, _, fl := newAPI(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, store.Site{Name: "One", URL: "https://one.com", Strategy: store.StrategyMobile})

	rec := do(t, a, http.MethodPost, "/backups", "")
	var info backupInfo
	decode(t, rec, &info)

	// mutate, then restore the earlier snapshot
	_, _ = s.CreateSite(ctx, store.Site{Name: "Two", URL: "https://two.com", Strategy: store.StrategyMobile})

	rec = do(t, a, http.MethodPost, "/backups/"+info.Name+"/restore", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("restore -> %d (body=%s)", rec.Code, rec.Body.String())
	}
	sites, _ := s.ListSites(ctx)
	if len(sites) != 1 || sites[0].Name != "One" {
		t.Fatalf("after restore = %+v, want only One", sites)
	}
	if fl.reloads() < 1 {
		t.Errorf("expected scheduler reload after restore, got %d", fl.reloads())
	}
}

func TestRestoreUpload(t *testing.T) {
	a, s, _, _ := newAPI(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, store.Site{Name: "One", URL: "https://one.com", Strategy: store.StrategyMobile})

	// Produce a valid backup file to upload.
	rec := do(t, a, http.MethodPost, "/backups", "")
	var info backupInfo
	decode(t, rec, &info)
	data, err := os.ReadFile(filepath.Join(a.backupDir, info.Name))
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	_, _ = s.CreateSite(ctx, store.Site{Name: "Two", URL: "https://two.com", Strategy: store.StrategyMobile})

	// Build multipart upload.
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "restore.db")
	_, _ = fw.Write(data)
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/restore", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	a.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("restore upload -> %d (body=%s)", rr.Code, rr.Body.String())
	}
	sites, _ := s.ListSites(ctx)
	if len(sites) != 1 || sites[0].Name != "One" {
		t.Fatalf("after upload restore = %+v, want only One", sites)
	}
}

func TestRestoreUploadRejectsGarbage(t *testing.T) {
	a, _, _, _ := newAPI(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "x.db")
	_, _ = fw.Write([]byte("not a sqlite db"))
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/restore", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	a.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("garbage restore -> %d, want 400", rr.Code)
	}
}
