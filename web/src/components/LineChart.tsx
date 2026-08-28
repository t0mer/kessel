export interface Series {
  label: string;
  color: string;
  points: { x: number; y: number }[]; // x = epoch ms, y = 0..100
}

/** LineChart plots 0–100 score series over time as a hand-rolled SVG (no deps). */
export function LineChart({ series, height = 220 }: { series: Series[]; height?: number }) {
  const width = 720;
  const padL = 28;
  const padR = 12;
  const padT = 12;
  const padB = 22;
  const all = series.flatMap((s) => s.points);
  if (all.length < 2) {
    return <div className="flex h-40 items-center justify-center text-sm text-muted">Not enough data to chart yet.</div>;
  }
  const xs = all.map((p) => p.x);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const spanX = maxX - minX || 1;

  const sx = (x: number) => padL + ((x - minX) / spanX) * (width - padL - padR);
  const sy = (y: number) => padT + (1 - y / 100) * (height - padT - padB);

  const gridY = [0, 25, 50, 75, 100];

  return (
    <div className="overflow-x-auto">
      <svg viewBox={`0 0 ${width} ${height}`} width="100%" className="min-w-[520px]" role="img" aria-label="Score history">
        {gridY.map((g) => (
          <g key={g}>
            <line x1={padL} x2={width - padR} y1={sy(g)} y2={sy(g)} stroke="var(--color-border)" strokeWidth="1" />
            <text x={4} y={sy(g) + 3} className="font-mono" style={{ fontSize: 9, fill: "var(--color-muted)" }}>
              {g}
            </text>
          </g>
        ))}
        {series.map((s) => (
          <polyline
            key={s.label}
            fill="none"
            stroke={s.color}
            strokeWidth="2"
            strokeLinejoin="round"
            strokeLinecap="round"
            points={s.points.map((p) => `${sx(p.x).toFixed(1)},${sy(p.y).toFixed(1)}`).join(" ")}
          />
        ))}
      </svg>
      <div className="mt-2 flex flex-wrap gap-4">
        {series.map((s) => (
          <span key={s.label} className="inline-flex items-center gap-1.5 text-xs text-muted">
            <span className="inline-block h-2 w-3 rounded" style={{ background: s.color }} />
            {s.label}
          </span>
        ))}
      </div>
    </div>
  );
}
