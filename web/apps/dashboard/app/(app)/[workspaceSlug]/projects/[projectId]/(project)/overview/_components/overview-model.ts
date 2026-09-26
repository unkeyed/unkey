import type { ProjectOverview } from "@/lib/trpc/routers/deploy/project/overview";

export type ProjectShape = "empty" | "api" | "deploy" | "full";

export type OverviewModel = {
  shape: ProjectShape;
};

export function buildOverviewModel(data: ProjectOverview): OverviewModel {
  const hasApps = data.apps.length > 0;
  const hasApi = data.keyspaces.length > 0 || data.ratelimits.length > 0;
  const shape: ProjectShape =
    hasApps && hasApi ? "full" : hasApps ? "deploy" : hasApi ? "api" : "empty";

  return { shape };
}

export function ago(ms: number, now = Date.now()): string {
  const s = Math.max(0, Math.round((now - ms) / 1000));
  if (s < 60) {
    return "just now";
  }
  const m = Math.round(s / 60);
  if (m < 60) {
    return `${m}m ago`;
  }
  const h = Math.round(m / 60);
  if (h < 24) {
    return `${h}h ago`;
  }
  const d = Math.round(h / 24);
  if (d < 30) {
    return `${d}d ago`;
  }
  return `${Math.round(d / 30)}mo ago`;
}

export function compact(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 }).format(n);
}
