import { fmtScore, scoreColor } from "@/lib/format";

/**
 * Gauge is Kessel's signature radial score dial: a glowing arc that fills
 * proportionally to a 0–100 Lighthouse score, with the number read as a
 * mono instrument readout in the center.
 */
export function Gauge({
  value,
  label,
  size = 96,
}: {
  value: number | null | undefined;
  label?: string;
  size?: number;
}) {
  const stroke = Math.max(6, size * 0.08);
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const pct = value == null ? 0 : Math.max(0, Math.min(100, value)) / 100;
  const color = scoreColor(value);

  return (
    <div className="inline-flex flex-col items-center gap-1.5">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--color-surface-2)" strokeWidth={stroke} />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c * (1 - pct)}
          style={{ transition: "stroke-dashoffset 700ms ease", filter: `drop-shadow(0 0 6px ${color}55)` }}
        />
        <text
          x="50%"
          y="50%"
          dominantBaseline="central"
          textAnchor="middle"
          className="rotate-90 font-mono font-medium"
          style={{ fontSize: size * 0.28, fill: "var(--color-fg)", transformOrigin: "center" }}
        >
          {fmtScore(value)}
        </text>
      </svg>
      {label && <span className="text-xs font-medium uppercase tracking-wide text-muted">{label}</span>}
    </div>
  );
}
