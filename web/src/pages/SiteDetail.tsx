import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Play, Plus, Trash2, ExternalLink } from "lucide-react";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import { CATEGORY_LABELS, type Channel, type Run, type Schedule, type Site, type ThresholdCategory, type ThresholdMode, type ThresholdRule } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { LineChart, type Series } from "@/components/LineChart";
import { Badge, Button, Card, ErrorState, Field, Input, Modal, Select, Spinner, Toggle } from "@/components/ui";
import { fmtDate, fmtScore } from "@/lib/format";

export function SiteDetail() {
  const { id } = useParams();
  const siteId = Number(id);
  const { data: site, error, loading } = useApi<Site>(() => api.getSite(siteId), [siteId], 0);
  const [notice, setNotice] = useState<string>();

  if (loading) return <Spinner />;
  if (error) return <ErrorState message={error} />;
  if (!site) return null;

  async function runNow() {
    try {
      await api.runSite(siteId);
      setNotice("Run started. Scores will appear here once it finishes.");
    } catch (e) {
      setNotice(e instanceof Error ? e.message : "Run failed to start.");
    }
  }

  return (
    <>
      <PageHeader
        title={site.name}
        subtitle={site.url}
        action={
          <Button onClick={runNow}>
            <Play size={16} /> Run now
          </Button>
        }
      />
      {notice && <div className="mb-4 rounded-lg border border-primary/40 bg-primary/10 px-4 py-2 text-sm text-primary">{notice}</div>}
      <div className="grid gap-6">
        <HistorySection siteId={siteId} />
        <SchedulesSection siteId={siteId} />
        <ThresholdsSection siteId={siteId} />
        <NotificationsSection siteId={siteId} />
      </div>
    </>
  );
}

function Section({ title, action, children }: { title: string; action?: React.ReactNode; children: React.ReactNode }) {
  return (
    <Card className="p-5">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="font-display text-lg font-semibold text-fg">{title}</h2>
        {action}
      </div>
      {children}
    </Card>
  );
}

function HistorySection({ siteId }: { siteId: number }) {
  const { data } = useApi(() => api.listRuns({ site_id: siteId, limit: 100 }), [siteId], 0);
  const runs = (data?.items ?? []).filter((r) => r.status === "success");
  const byStrategy = (strat: string, color: string): Series => ({
    label: strat,
    color,
    points: runs
      .filter((r) => r.strategy === strat && r.scores.performance != null)
      .map((r) => ({ x: new Date(r.started_at).getTime(), y: r.scores.performance as number }))
      .reverse(),
  });
  const series = [byStrategy("mobile", "var(--color-primary)"), byStrategy("desktop", "#a78bfa")].filter(
    (s) => s.points.length > 0,
  );
  const latestReport = (data?.items ?? []).find((r) => r.report_path);

  return (
    <Section
      title="Performance history"
      action={
        latestReport ? (
          <a href={api.reportURL(latestReport.id)} target="_blank" rel="noreferrer">
            <Button size="sm" variant="outline">
              <ExternalLink size={15} /> Latest report
            </Button>
          </a>
        ) : undefined
      }
    >
      <LineChart series={series} />
      {runs.length > 0 && (
        <div className="mt-5 overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs uppercase tracking-wide text-muted">
                <th className="py-2 pr-4 font-medium">When</th>
                <th className="py-2 pr-4 font-medium">Strategy</th>
                <th className="py-2 pr-4 font-medium">Perf</th>
                <th className="py-2 pr-4 font-medium">A11y</th>
                <th className="py-2 pr-4 font-medium">BP</th>
                <th className="py-2 pr-4 font-medium">SEO</th>
                <th className="py-2 font-medium">Report</th>
              </tr>
            </thead>
            <tbody className="font-mono">
              {(data?.items ?? []).slice(0, 15).map((r: Run) => (
                <tr key={r.id} className="border-t border-border">
                  <td className="py-2 pr-4 whitespace-nowrap">{fmtDate(r.started_at)}</td>
                  <td className="py-2 pr-4">{r.strategy}</td>
                  <td className="py-2 pr-4">{r.status === "success" ? fmtScore(r.scores.performance) : "—"}</td>
                  <td className="py-2 pr-4">{fmtScore(r.scores.accessibility)}</td>
                  <td className="py-2 pr-4">{fmtScore(r.scores.best_practices)}</td>
                  <td className="py-2 pr-4">{fmtScore(r.scores.seo)}</td>
                  <td className="py-2">
                    {r.report_path ? (
                      <a className="text-primary hover:underline" href={api.reportURL(r.id)} target="_blank" rel="noreferrer">
                        view
                      </a>
                    ) : r.status === "error" ? (
                      <Badge tone="poor">failed</Badge>
                    ) : (
                      "—"
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Section>
  );
}

function SchedulesSection({ siteId }: { siteId: number }) {
  const { data, reload } = useApi<Schedule[]>(() => api.listSchedules(siteId), [siteId], 0);
  const [cron, setCron] = useState("");
  const [err, setErr] = useState<string>();

  async function add() {
    setErr(undefined);
    try {
      await api.createSchedule(siteId, cron.trim());
      setCron("");
      reload();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Failed to add schedule.");
    }
  }

  return (
    <Section title="Schedules">
      <div className="flex flex-wrap gap-2">
        <Input value={cron} onChange={(e) => setCron(e.target.value)} placeholder="@every 6h  or  0 */4 * * *" className="max-w-xs" />
        <Button onClick={add} disabled={!cron.trim()}>
          <Plus size={16} /> Add
        </Button>
      </div>
      {err && <div className="mt-2"><ErrorState message={err} /></div>}
      <div className="mt-4 space-y-2">
        {(data ?? []).length === 0 && <p className="text-sm text-muted">No schedules. This site only runs when triggered manually.</p>}
        {(data ?? []).map((sc) => (
          <div key={sc.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <code className="font-mono text-sm text-fg">{sc.cron_expr}</code>
            <div className="flex items-center gap-3">
              <Toggle
                checked={sc.enabled}
                label="Enabled"
                onChange={async (v) => {
                  await api.updateSchedule(sc.id, { cron_expr: sc.cron_expr, enabled: v });
                  reload();
                }}
              />
              <Button size="sm" variant="ghost" aria-label="Delete schedule" onClick={async () => { await api.deleteSchedule(sc.id); reload(); }}>
                <Trash2 size={15} />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </Section>
  );
}

function ThresholdsSection({ siteId }: { siteId: number }) {
  const { data, reload } = useApi<ThresholdRule[]>(() => api.listThresholds(siteId), [siteId], 0);
  const [open, setOpen] = useState(false);

  return (
    <Section
      title="Threshold rules"
      action={
        <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
          <Plus size={15} /> Add rule
        </Button>
      }
    >
      <div className="space-y-2">
        {(data ?? []).length === 0 && <p className="text-sm text-muted">No rules. Add one to be notified when a score drops.</p>}
        {(data ?? []).map((r) => (
          <div key={r.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
            <span className="text-fg">
              <span className="font-medium">{CATEGORY_LABELS[r.category]}</span>{" "}
              <span className="text-muted">
                {r.mode === "absolute" ? `below ${r.value}` : `drops more than ${r.value} pts`}
              </span>
            </span>
            <div className="flex items-center gap-3">
              <Toggle
                checked={r.enabled}
                label="Enabled"
                onChange={async (v) => {
                  await api.updateThreshold(r.id, { category: r.category, mode: r.mode, value: r.value, enabled: v });
                  reload();
                }}
              />
              <Button size="sm" variant="ghost" aria-label="Delete rule" onClick={async () => { await api.deleteThreshold(r.id); reload(); }}>
                <Trash2 size={15} />
              </Button>
            </div>
          </div>
        ))}
      </div>
      {open && <ThresholdForm siteId={siteId} onClose={() => setOpen(false)} onSaved={() => { setOpen(false); reload(); }} />}
    </Section>
  );
}

function NotificationsSection({ siteId }: { siteId: number }) {
  const { data: channels } = useApi<Channel[]>(() => api.listChannels(), [], 0);
  const { data: linked, reload } = useApi(() => api.getSiteChannels(siteId), [siteId], 0);
  const linkedSet = new Set(linked?.channel_ids ?? []);

  async function toggle(id: number, on: boolean) {
    const next = new Set(linkedSet);
    if (on) next.add(id);
    else next.delete(id);
    await api.setSiteChannels(siteId, [...next]);
    reload();
  }

  return (
    <Section
      title="Notifications"
      action={
        <Link to="/channels">
          <Button size="sm" variant="outline">Manage channels</Button>
        </Link>
      }
    >
      {(channels ?? []).length === 0 ? (
        <p className="text-sm text-muted">
          No channels defined yet.{" "}
          <Link to="/channels" className="text-primary hover:underline">Add one</Link> to alert this site.
        </p>
      ) : (
        <>
          <p className="mb-3 text-sm text-muted">Choose which channels receive alerts for this site.</p>
          <div className="space-y-2">
            {(channels ?? []).map((c) => (
              <div key={c.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
                <span className="text-fg">
                  {c.name} <span className="text-muted">· {c.type}</span>
                  {!c.enabled && <span className="ml-2 text-xs text-muted">(disabled)</span>}
                </span>
                <Toggle checked={linkedSet.has(c.id)} label={`Notify ${c.name}`} onChange={(v) => toggle(c.id, v)} />
              </div>
            ))}
          </div>
        </>
      )}
    </Section>
  );
}

function ThresholdForm({ siteId, onClose, onSaved }: { siteId: number; onClose: () => void; onSaved: () => void }) {
  const [category, setCategory] = useState<ThresholdCategory>("performance");
  const [mode, setMode] = useState<ThresholdMode>("absolute");
  const [value, setValue] = useState("80");
  const [err, setErr] = useState<string>();

  async function save() {
    setErr(undefined);
    try {
      await api.createThreshold(siteId, { category, mode, value: Number(value) });
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Save failed.");
    }
  }

  return (
    <Modal title="Add threshold rule" onClose={onClose}>
      <div className="space-y-4">
        <Field label="Category">
          <Select value={category} onChange={(e) => setCategory(e.target.value as ThresholdCategory)}>
            {Object.entries(CATEGORY_LABELS).map(([k, v]) => (
              <option key={k} value={k}>{v}</option>
            ))}
          </Select>
        </Field>
        <Field label="Mode" hint="Absolute: score below the value. Delta: score dropped more than the value vs the previous run.">
          <Select value={mode} onChange={(e) => setMode(e.target.value as ThresholdMode)}>
            <option value="absolute">Absolute</option>
            <option value="delta">Delta (drop)</option>
          </Select>
        </Field>
        <Field label="Value">
          <Input type="number" value={value} onChange={(e) => setValue(e.target.value)} />
        </Field>
        {err && <ErrorState message={err} />}
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button onClick={save}>Save</Button>
        </div>
      </div>
    </Modal>
  );
}
