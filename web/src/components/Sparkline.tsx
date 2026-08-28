/**
 * Sparkline draws a compact SVG trend line for a series of 0–100 scores
 * (oldest→newest). Hand-rolled to keep the bundle dependency-free and offline.
 */
export function Sparkline({
  values,
  width = 120,
  height = 32,
  color = "var(--color-primary)",
}: {
  values: number[];
  width?: number;
  height?: number;
  color?: string;
}) {
  if (values.length < 2) {
    return <div style={{ width, height }} className="flex items-center text-xs text-muted">no trend</div>;
  }
  const pad = 2;
  const min = 0;
  const max = 100;
  const stepX = (width - pad * 2) / (values.length - 1);
  const points = values.map((v, i) => {
    const x = pad + i * stepX;
    const norm = (Math.max(min, Math.min(max, v)) - min) / (max - min);
    const y = height - pad - norm * (height - pad * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const last = values[values.length - 1];
  const lastNorm = (Math.max(min, Math.min(max, last)) - min) / (max - min);
  const lastY = height - pad - lastNorm * (height - pad * 2);

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
      <polyline points={points.join(" ")} fill="none" stroke={color} strokeWidth="1.75" strokeLinejoin="round" strokeLinecap="round" />
      <circle cx={pad + (values.length - 1) * stepX} cy={lastY} r="2.5" fill={color} />
    </svg>
  );
}
