import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import type { Run, Site } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { Gauge } from "@/components/Gauge";
import { Sparkline } from "@/components/Sparkline";
import { Badge, Card, EmptyState, ErrorState, Spinner, Button } from "@/components/ui";
import { fmtRelative } from "@/lib/format";

export function Dashboard() {
  const { data: sites, error, loading } = useApi<Site[]>(() => api.listSites(), []);

  return (
    <>
      <PageHeader
        title="Dashboard"
        subtitle="Latest PageSpeed scores across your sites"
        action={
          <Link to="/sites">
            <Button variant="outline">Manage sites</Button>
          </Link>
        }
      />
      {loading && <Spinner />}
      {error && <ErrorState message={error} />}
      {sites && sites.length === 0 && (
        <EmptyState
          title="No sites yet"
          hint="Add a site to start tracking its performance."
          action={
            <Link to="/sites">
              <Button>Add a site</Button>
            </Link>
          }
        />
      )}
      {sites && sites.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2">
          {sites.map((s) => (
            <SiteCard key={s.id} site={s} />
          ))}
        </div>
      )}
    </>
  );
}

function SiteCard({ site }: { site: Site }) {
  const { data } = useApi(() => api.listRuns({ site_id: site.id, limit: 20 }), [site.id]);
  const runs = data?.items ?? [];
  const latest = runs.find((r) => r.status === "success") ?? runs[0];
  const trend = runs
    .filter((r) => r.status === "success" && r.scores.performance != null)
    .slice(0, 12)
    .reverse()
    .map((r) => r.scores.performance as number);

  return (
    <Card className="p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <Link to={`/sites/${site.id}`} className="font-display text-lg font-semibold text-fg hover:text-primary">
            {site.name}
          </Link>
          <div className="truncate text-sm text-muted">{site.url}</div>
        </div>
        {site.enabled ? <Badge tone="primary">{site.strategy}</Badge> : <Badge>disabled</Badge>}
      </div>

      <div className="mt-4 flex items-center gap-5">
        <Gauge value={latest?.scores.performance ?? null} label="Performance" />
        <div className="flex-1">
          {trend.length >= 2 ? (
            <Sparkline values={trend} width={160} height={40} />
          ) : (
            <p className="text-sm text-muted">Not enough runs for a trend yet.</p>
          )}
          <div className="mt-2 text-xs text-muted">
            {latest ? <RunStatusLine run={latest} /> : "No runs yet"}
          </div>
        </div>
      </div>
    </Card>
  );
}

function RunStatusLine({ run }: { run: Run }) {
  return (
    <span className="inline-flex items-center gap-2">
      {run.status === "success" ? (
        <Badge tone="good">success</Badge>
      ) : (
        <Badge tone="poor">failed</Badge>
      )}
      <span>{fmtRelative(run.started_at)}</span>
    </span>
  );
}
