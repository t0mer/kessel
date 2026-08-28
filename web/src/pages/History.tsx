import { useState } from "react";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import type { Site } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { Badge, Card, ErrorState, Select, Spinner, Button } from "@/components/ui";
import { fmtDate, fmtScore } from "@/lib/format";

const PAGE = 25;

export function HistoryPage() {
  const { data: sites } = useApi<Site[]>(() => api.listSites(), [], 0);
  const [siteId, setSiteId] = useState<string>("");
  const [strategy, setStrategy] = useState<string>("");
  const [offset, setOffset] = useState(0);

  const { data, error, loading } = useApi(
    () =>
      api.listRuns({
        site_id: siteId ? Number(siteId) : undefined,
        strategy: strategy || undefined,
        limit: PAGE,
        offset,
      }),
    [siteId, strategy, offset],
    0,
  );

  const total = data?.total ?? 0;

  return (
    <>
      <PageHeader title="History" subtitle="Every check Kessel has run" />
      <Card className="mb-4 flex flex-wrap items-center gap-3 p-4">
        <Select
          value={siteId}
          onChange={(e) => { setSiteId(e.target.value); setOffset(0); }}
          className="max-w-xs"
        >
          <option value="">All sites</option>
          {(sites ?? []).map((s) => (
            <option key={s.id} value={s.id}>{s.name}</option>
          ))}
        </Select>
        <Select value={strategy} onChange={(e) => { setStrategy(e.target.value); setOffset(0); }} className="max-w-[10rem]">
          <option value="">All strategies</option>
          <option value="mobile">Mobile</option>
          <option value="desktop">Desktop</option>
        </Select>
      </Card>

      {loading && <Spinner />}
      {error && <ErrorState message={error} />}
      {data && (
        <Card className="overflow-x-auto p-0">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs uppercase tracking-wide text-muted">
                <th className="px-4 py-3 font-medium">When</th>
                <th className="px-4 py-3 font-medium">Strategy</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Perf</th>
                <th className="px-4 py-3 font-medium">A11y</th>
                <th className="px-4 py-3 font-medium">BP</th>
                <th className="px-4 py-3 font-medium">SEO</th>
                <th className="px-4 py-3 font-medium">Report</th>
              </tr>
            </thead>
            <tbody className="font-mono">
              {data.items.map((r) => (
                <tr key={r.id} className="border-t border-border">
                  <td className="whitespace-nowrap px-4 py-3">{fmtDate(r.started_at)}</td>
                  <td className="px-4 py-3">{r.strategy}</td>
                  <td className="px-4 py-3">
                    {r.status === "success" ? <Badge tone="good">success</Badge> : <Badge tone="poor">failed</Badge>}
                  </td>
                  <td className="px-4 py-3">{fmtScore(r.scores.performance)}</td>
                  <td className="px-4 py-3">{fmtScore(r.scores.accessibility)}</td>
                  <td className="px-4 py-3">{fmtScore(r.scores.best_practices)}</td>
                  <td className="px-4 py-3">{fmtScore(r.scores.seo)}</td>
                  <td className="px-4 py-3">
                    {r.report_path ? (
                      <a className="text-primary hover:underline" href={api.reportURL(r.id)} target="_blank" rel="noreferrer">view</a>
                    ) : "—"}
                  </td>
                </tr>
              ))}
              {data.items.length === 0 && (
                <tr>
                  <td colSpan={8} className="px-4 py-10 text-center text-muted">No runs match these filters.</td>
                </tr>
              )}
            </tbody>
          </table>
        </Card>
      )}

      {total > PAGE && (
        <div className="mt-4 flex items-center justify-between text-sm text-muted">
          <span>
            {offset + 1}–{Math.min(offset + PAGE, total)} of {total}
          </span>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE))}>
              Previous
            </Button>
            <Button size="sm" variant="outline" disabled={offset + PAGE >= total} onClick={() => setOffset(offset + PAGE)}>
              Next
            </Button>
          </div>
        </div>
      )}
    </>
  );
}
