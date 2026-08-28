export function scoreClass(v: number | null | undefined): "good" | "avg" | "poor" | "na" {
  if (v == null) return "na";
  if (v >= 90) return "good";
  if (v >= 50) return "avg";
  return "poor";
}

export function scoreColor(v: number | null | undefined): string {
  switch (scoreClass(v)) {
    case "good":
      return "var(--good)";
    case "avg":
      return "var(--avg)";
    case "poor":
      return "var(--poor)";
    default:
      return "var(--muted)";
  }
}

export function fmtScore(v: number | null | undefined): string {
  return v == null ? "—" : String(Math.round(v));
}

export function fmtMs(v: number | null | undefined): string {
  if (v == null) return "—";
  if (v >= 1000) return (v / 1000).toFixed(1) + " s";
  return Math.round(v) + " ms";
}

export function fmtNum(v: number | null | undefined, digits = 3): string {
  return v == null ? "—" : v.toFixed(digits);
}

export function fmtDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function fmtRelative(iso: string): string {
  const d = new Date(iso).getTime();
  if (isNaN(d)) return iso;
  const diff = Date.now() - d;
  const mins = Math.round(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.round(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.round(hrs / 24);
  return `${days}d ago`;
}
