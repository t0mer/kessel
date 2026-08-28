package api

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/kessel/internal/crypto"
)

const (
	backupPrefix  = "kessel-backup-"
	backupSuffix  = ".zip"
	dbEntry       = "kessel.db"
	keyEntry      = "kessel.key"
	maxUploadSize = 2 << 30 // 2 GiB: cap on an uploaded file AND on a decompressed db entry
	maxKeyBytes   = 4 << 10 // key entry is a 64-char hex string; cap generously
)

type backupInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

// safeBackupPath resolves a backup name to a path inside backupDir, rejecting
// traversal and unexpected names.
func (a *API) safeBackupPath(name string) (string, bool) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return "", false
	}
	if !strings.HasPrefix(name, backupPrefix) || !strings.HasSuffix(name, backupSuffix) {
		return "", false
	}
	return filepath.Join(a.backupDir, name), true
}

func (a *API) createBackup(w http.ResponseWriter, r *http.Request) {
	if err := os.MkdirAll(a.backupDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "creating backup directory")
		return
	}
	base := backupPrefix + time.Now().UTC().Format("20060102T150405Z")
	name := base + backupSuffix
	dest := filepath.Join(a.backupDir, name)
	for i := 2; fileExists(dest); i++ { // avoid collisions within the same second
		name = fmt.Sprintf("%s-%d%s", base, i, backupSuffix)
		dest = filepath.Join(a.backupDir, name)
	}
	if err := a.writeBackupArchive(r.Context(), dest); err != nil {
		_ = os.Remove(dest)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	fi, err := os.Stat(dest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reading backup")
		return
	}
	writeJSON(w, http.StatusCreated, backupInfo{Name: name, Size: fi.Size(), CreatedAt: time.Now().UTC().Format(time.RFC3339)})
}

// writeBackupArchive snapshots the DB and writes a zip containing the DB plus
// the encryption key (so the archive can be restored on another install).
func (a *API) writeBackupArchive(ctx context.Context, dest string) error {
	tf, err := os.CreateTemp(a.backupDir, "snapshot-*.db")
	if err != nil {
		return fmt.Errorf("staging snapshot: %w", err)
	}
	snap := tf.Name()
	_ = tf.Close()
	_ = os.Remove(snap) // VACUUM INTO requires the destination not to exist
	defer func() {
		_ = os.Remove(snap)
		_ = os.Remove(snap + "-wal")
		_ = os.Remove(snap + "-shm")
	}()
	if err := a.store.Backup(ctx, snap); err != nil {
		return err
	}

	// 0600: the archive bundles the encryption key, so it is as sensitive as
	// unencrypted data and must not be world-readable.
	zf, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating archive: %w", err)
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)

	dbw, err := zw.Create(dbEntry)
	if err != nil {
		return fmt.Errorf("writing archive db: %w", err)
	}
	snapFile, err := os.Open(snap)
	if err != nil {
		return fmt.Errorf("reading snapshot: %w", err)
	}
	if _, err := io.Copy(dbw, snapFile); err != nil {
		_ = snapFile.Close()
		return fmt.Errorf("archiving db: %w", err)
	}
	_ = snapFile.Close()

	kw, err := zw.Create(keyEntry)
	if err != nil {
		return fmt.Errorf("writing archive key: %w", err)
	}
	if _, err := kw.Write([]byte(a.keys.KeyHex())); err != nil {
		return fmt.Errorf("archiving key: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("finalizing archive: %w", err)
	}
	return nil
}

func (a *API) listBackups(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(a.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []backupInfo{})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]backupInfo, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasPrefix(n, backupPrefix) || !strings.HasSuffix(n, backupSuffix) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, backupInfo{Name: n, Size: fi.Size(), CreatedAt: fi.ModTime().UTC().Format(time.RFC3339)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name }) // newest first
	writeJSON(w, http.StatusOK, out)
}

func (a *API) downloadBackup(w http.ResponseWriter, r *http.Request) {
	path, ok := a.safeBackupPath(chi.URLParam(r, "name"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid backup name")
		return
	}
	if !fileExists(path) {
		writeError(w, http.StatusNotFound, "backup not found")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func (a *API) deleteBackup(w http.ResponseWriter, r *http.Request) {
	path, ok := a.safeBackupPath(chi.URLParam(r, "name"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid backup name")
		return
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "backup not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) restoreFromBackup(w http.ResponseWriter, r *http.Request) {
	path, ok := a.safeBackupPath(chi.URLParam(r, "name"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid backup name")
		return
	}
	if !fileExists(path) {
		writeError(w, http.StatusNotFound, "backup not found")
		return
	}
	a.doRestore(w, r, path)
}

func (a *API) restoreUpload(w http.ResponseWriter, r *http.Request) {
	if err := os.MkdirAll(a.backupDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "creating backup directory")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload (or file too large)")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	tmp, err := os.CreateTemp(a.backupDir, "restore-upload-*.tmp")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "staging upload")
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		writeError(w, http.StatusInternalServerError, "reading upload")
		return
	}
	_ = tmp.Close()

	a.doRestore(w, r, tmpPath)
}

func (a *API) doRestore(w http.ResponseWriter, r *http.Request, path string) {
	if err := a.restoreArchive(r.Context(), path); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Reflect restored schedules in the live scheduler.
	if err := a.reloader.Reload(r.Context()); err != nil {
		a.log.Error("reload after restore", "error", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

// restoreArchive restores from a Kessel backup .zip (db + key) — adopting the
// archived key — or, for backward compatibility, from a raw SQLite .db file
// (keeping the current key).
func (a *API) restoreArchive(ctx context.Context, path string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		// Not a zip: treat as a raw .db and keep the current key.
		return a.store.RestoreFrom(ctx, path)
	}
	defer zr.Close()

	var dbTmp, keyHex string
	for _, f := range zr.File {
		switch f.Name {
		case dbEntry:
			dbTmp, err = a.extractToTemp(f)
			if err != nil {
				return err
			}
		case keyEntry:
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("reading archived key: %w", err)
			}
			b, err := io.ReadAll(io.LimitReader(rc, maxKeyBytes))
			_ = rc.Close()
			if err != nil {
				return fmt.Errorf("reading archived key: %w", err)
			}
			keyHex = strings.TrimSpace(string(b))
		}
	}
	if dbTmp == "" {
		return fmt.Errorf("archive is missing %s", dbEntry)
	}
	defer os.Remove(dbTmp)

	if err := a.store.RestoreFrom(ctx, dbTmp); err != nil {
		return err
	}
	if keyHex != "" {
		key, err := crypto.ParseKey(keyHex)
		if err != nil {
			return fmt.Errorf("archive contains an invalid encryption key: %w", err)
		}
		if err := a.keys.Adopt(key); err != nil {
			return fmt.Errorf("adopting archived key: %w", err)
		}
	}
	return nil
}

func (a *API) extractToTemp(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", f.Name, err)
	}
	defer rc.Close()
	out, err := os.CreateTemp(a.backupDir, "restore-db-*.db")
	if err != nil {
		return "", fmt.Errorf("staging %s: %w", f.Name, err)
	}
	// Bound the *decompressed* size to guard against a zip bomb: a small upload
	// could otherwise inflate to fill the disk.
	n, err := io.Copy(out, io.LimitReader(rc, maxUploadSize+1))
	if err == nil && n > maxUploadSize {
		err = fmt.Errorf("archive entry %s exceeds %d bytes", f.Name, int64(maxUploadSize))
	}
	if err != nil {
		_ = out.Close()
		_ = os.Remove(out.Name())
		return "", fmt.Errorf("extracting %s: %w", f.Name, err)
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return out.Name(), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
