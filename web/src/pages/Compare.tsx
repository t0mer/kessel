import { useState } from "react";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import type { CompareResult, Run } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { Button, Card, ErrorState, Field, Select } from "@/components/ui";
import { fmtDate, fmtScore, fmtMs } from "@/lib/format";

export function Compare() {
  const { data } = useApi(() => api.listRuns({ limit: 100 }), [], 0);
  const runs = data?.items ?? [];
  const [a, setA] = useState("");
  const [b, setB] = useState("");
  const [result, setResult] = useState<CompareResult>();
  const [err, setErr] = useState<string>();

  async function run() {
    setErr(undefined);
    setResult(undefined);
    try {
      setResult(await api.compare(Number(a), Number(b)));
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Compare failed.");
    }
  }

  const label = (r: Run) => `#${r.id} · ${r.strategy} · ${fmtDate(r.started_at)}`;

  return (
    <>
      <PageHeader title="Compare runs" subtitle="See how two checks differ, side by side" />
      <Card className="mb-6 p-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Run A (baseline)">
            <Select value={a} onChange={(e) => setA(e.target.value)}>
              <option value="">Select a run…</option>
              {runs.map((r) => (
                <option key={r.id} value={r.id}>{label(r)}</option>
              ))}
            </Select>
          </Field>
          <Field label="Run B (comparison)">
            <Select value={b} onChange={(e) => setB(e.target.value)}>
              <option value="">Select a run…</option>
              {runs.map((r) => (
                <option key={r.id} value={r.id}>{label(r)}</option>
              ))}
            </Select>
          </Field>
        </div>
        <div className="mt-4">
          <Button onClick={run} disabled={!a || !b || a === b}>Compare</Button>
        </div>
        {err && <div className="mt-3"><ErrorState message={err} /></div>}
      </Card>

      {result && <CompareTable result={result} />}
    </>
  );
}

function CompareTable({ result }: { result: CompareResult }) {
  const scoreRows: { label: string; key: keyof CompareResult["score_deltas"] }[] = [
    { label: "Performance", key: "performance" },
    { label: "Accessibility", key: "accessibility" },
    { label: "Best Practices", key: "best_practices" },
    { label: "SEO", key: "seo" },
  ];
  const metricRows: { label: string; key: keyof CompareResult["metric_deltas"]; ms?: boolean }[] = [
    { label: "LCP", key: "lcp_ms", ms: true },
    { label: "CLS", key: "cls" },
    { label: "TBT", key: "tbt_ms", ms: true },
    { label: "FCP", key: "fcp_ms", ms: true },
    { label: "Speed Index", key: "si_ms", ms: true },
    { label: "TTI", key: "tti_ms", ms: true },
  ];

  return (
    <Card className="overflow-x-auto p-5">
      <div className="mb-4 grid grid-cols-3 gap-2 text-sm">
        <div />
        <div className="text-center text-muted">A · #{result.a.id}<br />{fmtDate(result.a.started_at)}</div>
        <div className="text-center text-muted">B · #{result.b.id}<br />{fmtDate(result.b.started_at)}</div>
      </div>
      <table className="w-full text-sm">
        <tbody className="font-mono">
          <tr><td colSpan={4} className="pb-2 pt-1 font-sans text-xs uppercase tracking-wide text-muted">Scores (higher is better)</td></tr>
          {scoreRows.map((row) => {
            const av = result.a.scores[row.key];
            const bv = result.b.scores[row.key];
            const d = result.score_deltas[row.key];
            return (
              <tr key={row.key} className="border-t border-border">
                <td className="py-2 font-sans text-fg">{row.label}</td>
                <td className="py-2 text-center">{fmtScore(av)}</td>
                <td className="py-2 text-center">{fmtScore(bv)}</td>
                <td className="py-2 pl-4 text-right">{delta(d, true)}</td>
              </tr>
            );
          })}
          <tr><td colSpan={4} className="pb-2 pt-4 font-sans text-xs uppercase tracking-wide text-muted">Metrics (lower is better)</td></tr>
          {metricRows.map((row) => {
            const av = result.a.metrics[row.key];
            const bv = result.b.metrics[row.key];
            const d = result.metric_deltas[row.key];
            const fmt = row.ms ? fmtMs : (v: number | null | undefined) => (v == null ? "—" : v.toFixed(3));
            return (
              <tr key={row.key} className="border-t border-border">
                <td className="py-2 font-sans text-fg">{row.label}</td>
                <td className="py-2 text-center">{fmt(av)}</td>
                <td className="py-2 text-center">{fmt(bv)}</td>
                <td className="py-2 pl-4 text-right">{delta(d, false)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </Card>
  );
}

// delta renders B−A. higherIsBetter flips the color meaning.
function delta(d: number | null | undefined, higherIsBetter: boolean) {
  if (d == null) return <span className="text-muted">—</span>;
  if (Math.abs(d) < 1e-9) return <span className="text-muted">0</span>;
  const improved = higherIsBetter ? d > 0 : d < 0;
  const sign = d > 0 ? "+" : "";
  return <span style={{ color: improved ? "var(--good)" : "var(--poor)" }}>{sign}{Number.isInteger(d) ? d : d.toFixed(2)}</span>;
}
