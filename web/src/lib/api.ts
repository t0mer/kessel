import type {
  Channel,
  CompareResult,
  Run,
  RunsPage,
  Schedule,
  Site,
  ThresholdRule,
} from "./types";

const BASE = "/api/v1";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    let message = `request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      /* non-JSON error body */
    }
    throw new Error(message);
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return text ? (JSON.parse(text) as T) : (undefined as T);
}

export const api = {
  // Sites
  listSites: () => request<Site[]>("/sites"),
  getSite: (id: number) => request<Site>(`/sites/${id}`),
  createSite: (body: Partial<Site>) =>
    request<Site>("/sites", { method: "POST", body: JSON.stringify(body) }),
  updateSite: (id: number, body: Partial<Site>) =>
    request<Site>(`/sites/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteSite: (id: number) => request<void>(`/sites/${id}`, { method: "DELETE" }),
  runSite: (id: number) => request<{ status: string }>(`/sites/${id}/run`, { method: "POST" }),

  // Schedules
  listSchedules: (siteId: number) => request<Schedule[]>(`/sites/${siteId}/schedules`),
  createSchedule: (siteId: number, cron_expr: string) =>
    request<Schedule>(`/sites/${siteId}/schedules`, {
      method: "POST",
      body: JSON.stringify({ cron_expr }),
    }),
  updateSchedule: (id: number, body: Partial<Schedule>) =>
    request<Schedule>(`/schedules/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteSchedule: (id: number) => request<void>(`/schedules/${id}`, { method: "DELETE" }),

  // Runs
  listRuns: (params: { site_id?: number; strategy?: string; limit?: number; offset?: number } = {}) => {
    const q = new URLSearchParams();
    if (params.site_id) q.set("site_id", String(params.site_id));
    if (params.strategy) q.set("strategy", params.strategy);
    if (params.limit) q.set("limit", String(params.limit));
    if (params.offset) q.set("offset", String(params.offset));
    const qs = q.toString();
    return request<RunsPage>(`/runs${qs ? "?" + qs : ""}`);
  },
  getRun: (id: number) => request<Run>(`/runs/${id}`),
  compare: (a: number, b: number) => request<CompareResult>(`/compare?a=${a}&b=${b}`),
  reportURL: (id: number) => `${BASE}/runs/${id}/report`,

  // Channels
  listChannels: () => request<Channel[]>("/channels"),
  createChannel: (body: unknown) =>
    request<Channel>("/channels", { method: "POST", body: JSON.stringify(body) }),
  updateChannel: (id: number, body: unknown) =>
    request<Channel>(`/channels/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteChannel: (id: number) => request<void>(`/channels/${id}`, { method: "DELETE" }),
  testChannel: (body: unknown) =>
    request<{ status: string }>("/channels/test", { method: "POST", body: JSON.stringify(body) }),

  // Per-site channel links
  getSiteChannels: (siteId: number) => request<{ channel_ids: number[] }>(`/sites/${siteId}/channels`),
  setSiteChannels: (siteId: number, channelIds: number[]) =>
    request<{ channel_ids: number[] }>(`/sites/${siteId}/channels`, {
      method: "PUT",
      body: JSON.stringify({ channel_ids: channelIds }),
    }),

  // Thresholds
  listThresholds: (siteId: number) => request<ThresholdRule[]>(`/sites/${siteId}/thresholds`),
  createThreshold: (siteId: number, body: Partial<ThresholdRule>) =>
    request<ThresholdRule>(`/sites/${siteId}/thresholds`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateThreshold: (id: number, body: Partial<ThresholdRule>) =>
    request<ThresholdRule>(`/thresholds/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteThreshold: (id: number) => request<void>(`/thresholds/${id}`, { method: "DELETE" }),
};
