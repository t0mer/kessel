package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	backupPrefix  = "kessel-backup-"
	backupSuffix  = ".db"
	maxUploadSize = 2 << 30 // 2 GiB
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
	if err := a.store.Backup(r.Context(), dest); err != nil {
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
	w.Header().Set("Content-Type", "application/octet-stream")
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
	if err := a.store.RestoreFrom(r.Context(), path); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Reflect restored schedules in the live scheduler.
	if err := a.reloader.Reload(r.Context()); err != nil {
		a.log.Error("reload after restore", "error", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
