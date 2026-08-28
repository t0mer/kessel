import { useState } from "react";
import { Link } from "react-router-dom";
import { Play, Pencil, Trash2, Plus } from "lucide-react";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import type { Site, Strategy } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { Badge, Button, Card, EmptyState, ErrorState, Field, Input, Modal, Select, Spinner, Toggle } from "@/components/ui";

export function Sites() {
  const { data: sites, error, loading, reload } = useApi<Site[]>(() => api.listSites(), [], 0);
  const [editing, setEditing] = useState<Site | null>(null);
  const [adding, setAdding] = useState(false);
  const [busy, setBusy] = useState<number | null>(null);
  const [notice, setNotice] = useState<string>();

  async function runNow(s: Site) {
    setBusy(s.id);
    try {
      await api.runSite(s.id);
      setNotice(`Run started for ${s.name}.`);
    } catch (e) {
      setNotice(e instanceof Error ? e.message : "Run failed to start.");
    } finally {
      setBusy(null);
    }
  }

  async function toggleEnabled(s: Site, enabled: boolean) {
    await api.updateSite(s.id, { name: s.name, url: s.url, strategy: s.strategy, enabled });
    reload();
  }

  async function remove(s: Site) {
    if (!confirm(`Delete ${s.name}? Its runs and reports will be removed.`)) return;
    await api.deleteSite(s.id);
    reload();
  }

  return (
    <>
      <PageHeader
        title="Sites"
        subtitle="Websites Kessel is monitoring"
        action={
          <Button onClick={() => setAdding(true)}>
            <Plus size={16} /> Add site
          </Button>
        }
      />
      {notice && <div className="mb-4 rounded-lg border border-primary/40 bg-primary/10 px-4 py-2 text-sm text-primary">{notice}</div>}
      {loading && <Spinner />}
      {error && <ErrorState message={error} />}
      {sites && sites.length === 0 && (
        <EmptyState title="No sites yet" hint="Add your first site to begin." action={<Button onClick={() => setAdding(true)}>Add site</Button>} />
      )}
      {sites && sites.length > 0 && (
        <div className="space-y-3">
          {sites.map((s) => (
            <Card key={s.id} className="flex flex-wrap items-center gap-4 p-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <Link to={`/sites/${s.id}`} className="font-medium text-fg hover:text-primary">
                    {s.name}
                  </Link>
                  <Badge tone="primary">{s.strategy}</Badge>
                </div>
                <div className="truncate text-sm text-muted">{s.url}</div>
              </div>
              <div className="flex items-center gap-3">
                <Toggle checked={s.enabled} onChange={(v) => toggleEnabled(s, v)} label="Enabled" />
                <Button size="sm" variant="outline" disabled={busy === s.id} onClick={() => runNow(s)}>
                  <Play size={15} /> {busy === s.id ? "Starting…" : "Run now"}
                </Button>
                <Button size="sm" variant="ghost" aria-label="Edit" onClick={() => setEditing(s)}>
                  <Pencil size={15} />
                </Button>
                <Button size="sm" variant="ghost" aria-label="Delete" onClick={() => remove(s)}>
                  <Trash2 size={15} />
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {(adding || editing) && (
        <SiteForm
          site={editing}
          onClose={() => {
            setAdding(false);
            setEditing(null);
          }}
          onSaved={() => {
            setAdding(false);
            setEditing(null);
            reload();
          }}
        />
      )}
    </>
  );
}

function SiteForm({ site, onClose, onSaved }: { site: Site | null; onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState(site?.name ?? "");
  const [url, setUrl] = useState(site?.url ?? "");
  const [strategy, setStrategy] = useState<Strategy>(site?.strategy ?? "mobile");
  const [err, setErr] = useState<string>();
  const [saving, setSaving] = useState(false);

  async function save() {
    setSaving(true);
    setErr(undefined);
    try {
      const body = { name, url, strategy, enabled: site?.enabled ?? true };
      if (site) await api.updateSite(site.id, body);
      else await api.createSite(body);
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Save failed.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal title={site ? "Edit site" : "Add site"} onClose={onClose}>
      <div className="space-y-4">
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Marketing site" />
        </Field>
        <Field label="URL" hint="Must be a full http or https URL.">
          <Input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com" />
        </Field>
        <Field label="Strategy">
          <Select value={strategy} onChange={(e) => setStrategy(e.target.value as Strategy)}>
            <option value="mobile">Mobile</option>
            <option value="desktop">Desktop</option>
            <option value="both">Both</option>
          </Select>
        </Field>
        {err && <ErrorState message={err} />}
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={save} disabled={saving}>
            {saving ? "Saving…" : "Save"}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
