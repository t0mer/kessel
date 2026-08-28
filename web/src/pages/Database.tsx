import { useRef, useState } from "react";
import { Download, DatabaseBackup, RotateCcw, Trash2, Upload } from "lucide-react";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import type { BackupInfo } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { Button, Card, EmptyState, ErrorState, Spinner } from "@/components/ui";
import { fmtBytes, fmtDate } from "@/lib/format";

function triggerDownload(url: string, filename: string) {
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
}

export function Database() {
  const { data: backups, loading, reload } = useApi<BackupInfo[]>(() => api.listBackups(), [], 0);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string>();
  const [err, setErr] = useState<string>();
  const fileRef = useRef<HTMLInputElement>(null);

  function flash(msg: string) {
    setNotice(msg);
    setErr(undefined);
  }
  function fail(e: unknown) {
    setErr(e instanceof Error ? e.message : String(e));
    setNotice(undefined);
  }

  async function backupNow() {
    setBusy(true);
    try {
      const info = await api.createBackup();
      triggerDownload(api.backupDownloadURL(info.name), info.name);
      flash(`Backup created and downloaded: ${info.name}`);
      reload();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function restore(name: string) {
    if (!confirm(`Restore from ${name}? This REPLACES the entire current database. This cannot be undone.`)) return;
    setBusy(true);
    try {
      await api.restoreFromBackup(name);
      flash("Database restored.");
      reload();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function remove(name: string) {
    if (!confirm(`Delete backup ${name}?`)) return;
    try {
      await api.deleteBackup(name);
      reload();
    } catch (e) {
      fail(e);
    }
  }

  async function restoreUpload(file: File) {
    if (!confirm(`Restore from ${file.name}? This REPLACES the entire current database. This cannot be undone.`)) return;
    setBusy(true);
    try {
      await api.restoreUpload(file);
      flash("Database restored from uploaded file.");
      reload();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
      if (fileRef.current) fileRef.current.value = "";
    }
  }

  return (
    <>
      <PageHeader
        title="Database"
        subtitle="Back up and restore the full Kessel database"
        action={
          <Button onClick={backupNow} disabled={busy}>
            <DatabaseBackup size={16} /> {busy ? "Working…" : "Back up now"}
          </Button>
        }
      />

      {notice && <div className="mb-4 rounded-lg border border-good/40 bg-good/10 px-4 py-2 text-sm text-good">{notice}</div>}
      {err && <div className="mb-4"><ErrorState message={err} /></div>}

      <Card className="mb-6 p-5">
        <h2 className="font-display text-lg font-semibold text-fg">Restore from a file</h2>
        <p className="mt-1 mb-4 text-sm text-muted">
          Upload a Kessel backup (<code className="font-mono">.db</code>) to replace the current database. Restoring a
          backup from a different install also needs that install's encryption key to read saved channel credentials.
        </p>
        <div className="flex flex-wrap items-center gap-3">
          <input
            ref={fileRef}
            type="file"
            accept=".db,application/octet-stream,application/x-sqlite3"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) restoreUpload(f);
            }}
            className="block text-sm text-muted file:mr-3 file:rounded-lg file:border file:border-border file:bg-surface-2 file:px-3 file:py-2 file:text-sm file:text-fg hover:file:bg-surface"
          />
          <span className="inline-flex items-center gap-1 text-xs text-muted"><Upload size={14} /> select a .db file</span>
        </div>
      </Card>

      <h2 className="mb-3 font-display text-lg font-semibold text-fg">Saved backups</h2>
      {loading && <Spinner />}
      {backups && backups.length === 0 && (
        <EmptyState title="No backups yet" hint="Backups are saved to the data folder and downloaded to your browser." action={<Button onClick={backupNow}>Back up now</Button>} />
      )}
      {backups && backups.length > 0 && (
        <div className="space-y-2">
          {backups.map((b) => (
            <Card key={b.name} className="flex flex-wrap items-center gap-4 p-4">
              <div className="min-w-0 flex-1">
                <div className="truncate font-mono text-sm text-fg">{b.name}</div>
                <div className="text-xs text-muted">{fmtBytes(b.size)} · {fmtDate(b.created_at)}</div>
              </div>
              <div className="flex items-center gap-2">
                <a href={api.backupDownloadURL(b.name)} download={b.name}>
                  <Button size="sm" variant="outline"><Download size={15} /> Download</Button>
                </a>
                <Button size="sm" variant="outline" disabled={busy} onClick={() => restore(b.name)}>
                  <RotateCcw size={15} /> Restore
                </Button>
                <Button size="sm" variant="ghost" aria-label="Delete backup" onClick={() => remove(b.name)}>
                  <Trash2 size={15} />
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  );
}
