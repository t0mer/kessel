import { useState } from "react";
import { ExternalLink } from "lucide-react";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import type { Run, Site } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { Card, EmptyState, ErrorState, Spinner } from "@/components/ui";
import { fmtDate } from "@/lib/format";

export function Reports() {
  const { data: sites } = useApi<Site[]>(() => api.listSites(), [], 0);
  const { data, error, loading } = useApi(() => api.listRuns({ limit: 100 }), [], 0);
  const withReports = (data?.items ?? []).filter((r) => r.report_path);
  const [selected, setSelected] = useState<Run | null>(null);

  const siteName = (id: number) => sites?.find((s) => s.id === id)?.name ?? `site ${id}`;

  return (
    <>
      <PageHeader title="Reports" subtitle="Full PageSpeed reports saved for each check" />
      {loading && <Spinner />}
      {error && <ErrorState message={error} />}
      {data && withReports.length === 0 && (
        <EmptyState title="No reports yet" hint="Reports are generated automatically after each successful check." />
      )}
      {withReports.length > 0 && (
        <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
          <div className="space-y-2">
            {withReports.map((r) => (
              <button
                key={r.id}
                onClick={() => setSelected(r)}
                className={[
                  "w-full rounded-lg border px-3 py-2.5 text-left transition-colors",
                  selected?.id === r.id ? "border-primary bg-primary/10" : "border-border bg-surface hover:bg-surface-2",
                ].join(" ")}
              >
                <div className="font-medium text-fg">{siteName(r.site_id)}</div>
                <div className="text-xs text-muted">{r.strategy} · {fmtDate(r.started_at)}</div>
              </button>
            ))}
          </div>
          <Card className="min-h-[60vh] overflow-hidden p-0">
            {selected ? (
              <div className="flex h-full flex-col">
                <div className="flex items-center justify-between border-b border-border px-4 py-2 text-sm">
                  <span className="text-muted">{siteName(selected.site_id)} · {selected.strategy}</span>
                  <a className="inline-flex items-center gap-1 text-primary hover:underline" href={api.reportURL(selected.id)} target="_blank" rel="noreferrer">
                    Open <ExternalLink size={14} />
                  </a>
                </div>
                <iframe
                  title="Report preview"
                  src={api.reportURL(selected.id)}
                  className="h-[70vh] w-full flex-1 bg-white"
                />
              </div>
            ) : (
              <div className="flex h-[60vh] items-center justify-center text-sm text-muted">Select a report to preview it here.</div>
            )}
          </Card>
        </div>
      )}
    </>
  );
}
