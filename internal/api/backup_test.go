package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/t0mer/kessel/internal/crypto"
	"github.com/t0mer/kessel/internal/store"
)

func TestCreateListDownloadBackup(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "One", URL: "https://one.com", Strategy: store.StrategyMobile})

	rec := do(t, a, http.MethodPost, "/backups", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create -> %d (body=%s)", rec.Code, rec.Body.String())
	}
	var info backupInfo
	decode(t, rec, &info)
	if info.Name == "" || info.Size == 0 || filepath.Ext(info.Name) != ".zip" {
		t.Fatalf("unexpected backup info: %+v", info)
	}

	rec = do(t, a, http.MethodGet, "/backups", "")
	var list []backupInfo
	decode(t, rec, &list)
	if len(list) != 1 || list[0].Name != info.Name {
		t.Fatalf("list = %+v", list)
	}

	rec = do(t, a, http.MethodGet, "/backups/"+info.Name, "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Disposition") == "" || rec.Body.Len() == 0 {
		t.Fatalf("download: code=%d cd=%q len=%d", rec.Code, rec.Header().Get("Content-Disposition"), rec.Body.Len())
	}
}

func TestBackupArchiveContainsDbAndKey(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "One", URL: "https://one.com", Strategy: store.StrategyMobile})
	rec := do(t, a, http.MethodPost, "/backups", "")
	var info backupInfo
	decode(t, rec, &info)

	archPath := filepath.Join(a.backupDir, info.Name)
	if fi, err := os.Stat(archPath); err != nil {
		t.Fatalf("stat archive: %v", err)
	} else if fi.Mode().Perm() != 0o600 {
		t.Errorf("archive perms = %v, want 0600 (contains the key)", fi.Mode().Perm())
	}

	zr, err := zip.OpenReader(archPath)
	if err != nil {
		t.Fatalf("archive is not a valid zip: %v", err)
	}
	defer zr.Close()
	names := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		names[f.Name] = string(b)
	}
	if _, ok := names["kessel.db"]; !ok {
		t.Error("archive missing kessel.db")
	}
	key, ok := names["kessel.key"]
	if !ok || len(key) != 64 {
		t.Errorf("archive key entry wrong: present=%v len=%d", ok, len(key))
	}
	if key != a.keysForTest().KeyHex() {
		t.Error("archived key does not match the live key")
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

	_, _ = s.CreateSite(ctx, store.Site{Name: "Two", URL: "https://two.com", Strategy: store.StrategyMobile})

	rec = do(t, a, http.MethodPost, "/backups/"+info.Name+"/restore", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("restore -> %d (body=%s)", rec.Code, rec.Body.String())
	}
	sites, _ := s.ListSites(ctx)
	if len(sites) != 1 || sites[0].Name != "One" {
		t.Fatalf("after restore = %+v", sites)
	}
	if fl.reloads() < 1 {
		t.Error("expected scheduler reload after restore")
	}
}

func TestRestoreAdoptsArchivedKey(t *testing.T) {
	a, s, _, _ := newAPI(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, store.Site{Name: "Local", URL: "https://local.com", Strategy: store.StrategyMobile})

	// Build a valid Kessel DB "from another install" and a fresh key.
	srcPath := filepath.Join(t.TempDir(), "src.db")
	src, err := store.Open(srcPath)
	if err != nil {
		t.Fatalf("open src: %v", err)
	}
	_, _ = src.CreateSite(ctx, store.Site{Name: "Imported", URL: "https://imported.com", Strategy: store.StrategyDesktop})
	_ = src.Close()

	newKey, _ := crypto.NewKey()
	archive := filepath.Join(t.TempDir(), "backup.zip")
	writeArchive(t, archive, srcPath, hex.EncodeToString(newKey))

	uploadRestore(t, a, archive)

	// DB replaced with the imported data.
	sites, _ := s.ListSites(ctx)
	if len(sites) != 1 || sites[0].Name != "Imported" {
		t.Fatalf("after restore = %+v, want only Imported", sites)
	}
	// Live key adopted from the archive.
	if a.keysForTest().KeyHex() != hex.EncodeToString(newKey) {
		t.Fatal("archived key was not adopted")
	}
}

func TestRestoreRawDbBackwardCompat(t *testing.T) {
	a, s, _, _ := newAPI(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, store.Site{Name: "One", URL: "https://one.com", Strategy: store.StrategyMobile})
	before := a.keysForTest().KeyHex()

	// A raw .db snapshot (no archive, no key).
	rawDB := filepath.Join(t.TempDir(), "raw.db")
	if err := s.Backup(ctx, rawDB); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	_, _ = s.CreateSite(ctx, store.Site{Name: "Two", URL: "https://two.com", Strategy: store.StrategyMobile})

	uploadRestore(t, a, rawDB)

	sites, _ := s.ListSites(ctx)
	if len(sites) != 1 || sites[0].Name != "One" {
		t.Fatalf("after raw restore = %+v", sites)
	}
	if a.keysForTest().KeyHex() != before {
		t.Fatal("raw .db restore should not change the key")
	}
}

func TestRestoreUploadRejectsGarbage(t *testing.T) {
	a, _, _, _ := newAPI(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "x.zip")
	_, _ = fw.Write([]byte("not a sqlite db and not a zip"))
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/restore", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	a.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("garbage restore -> %d, want 400", rr.Code)
	}
}

// --- helpers ---

func writeArchive(t *testing.T, dest, dbPath, keyHex string) {
	t.Helper()
	zf, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)
	dbw, _ := zw.Create("kessel.db")
	in, err := os.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dbw, in); err != nil {
		t.Fatal(err)
	}
	_ = in.Close()
	kw, _ := zw.Create("kessel.key")
	_, _ = kw.Write([]byte(keyHex))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func uploadRestore(t *testing.T, a *API, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", filepath.Base(path))
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/restore", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	a.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("restore upload -> %d (body=%s)", rr.Code, rr.Body.String())
	}
}
