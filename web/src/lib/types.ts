export type Strategy = "mobile" | "desktop" | "both";

export interface Site {
  id: number;
  name: string;
  slug: string;
  url: string;
  strategy: Strategy;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Schedule {
  id: number;
  site_id: number;
  cron_expr: string;
  enabled: boolean;
  created_at: string;
}

export interface Scores {
  performance: number | null;
  accessibility: number | null;
  best_practices: number | null;
  seo: number | null;
}

export interface Metrics {
  lcp_ms: number | null;
  cls: number | null;
  tbt_ms: number | null;
  fcp_ms: number | null;
  si_ms: number | null;
  tti_ms: number | null;
}

export type RunStatus = "success" | "error";

export interface Run {
  id: number;
  site_id: number;
  strategy: Exclude<Strategy, "both">;
  status: RunStatus;
  started_at: string;
  finished_at: string;
  scores: Scores;
  metrics: Metrics;
  report_path?: string;
  error?: string;
}

export interface RunsPage {
  items: Run[];
  total: number;
  limit: number;
  offset: number;
}

export interface CompareResult {
  a: Run;
  b: Run;
  score_deltas: Scores;
  metric_deltas: Metrics;
}

export type ChannelType = "shoutrrr" | "greenapi" | "whatsapp_web";

export interface Channel {
  id: number;
  type: ChannelType;
  name: string;
  enabled: boolean;
  notify_on_success: boolean;
  notify_on_failure: boolean;
}

export type ThresholdCategory =
  | "performance"
  | "accessibility"
  | "best_practices"
  | "seo";
export type ThresholdMode = "absolute" | "delta";

export interface ThresholdRule {
  id: number;
  site_id: number;
  category: ThresholdCategory;
  mode: ThresholdMode;
  value: number;
  enabled: boolean;
}

export const CATEGORY_LABELS: Record<string, string> = {
  performance: "Performance",
  accessibility: "Accessibility",
  best_practices: "Best Practices",
  seo: "SEO",
};
