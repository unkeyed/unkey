import { statusGroupOf } from "@/lib/collections/deploy/deployment-status";
import type {
  OverviewApp,
  OverviewDeployment,
  ProjectOverview,
} from "@/lib/trpc/routers/deploy/project/overview";

export type ProjectShape = "empty" | "api" | "deploy" | "full";

export type Attention =
  | { kind: "failed"; app: OverviewApp; deployment: OverviewDeployment }
  | { kind: "building"; app: OverviewApp; deployment: OverviewDeployment }
  | { kind: "awaiting"; app: OverviewApp; deployment: OverviewDeployment };

export type SetupStep = {
  id: "github" | "app" | "deploy" | "keyspace" | "ratelimit";
  label: string;
  done: boolean;
};

export type OverviewModel = {
  shape: ProjectShape;
  attention: Attention[];
  steps: SetupStep[];
  liveApps: number;
};

export function buildOverviewModel(data: ProjectOverview): OverviewModel {
  const hasApps = data.apps.length > 0;
  const hasApi = data.keyspaces.length > 0 || data.ratelimits.length > 0;
  const shape: ProjectShape =
    hasApps && hasApi ? "full" : hasApps ? "deploy" : hasApi ? "api" : "empty";

  const attention: Attention[] = [];
  for (const app of data.apps) {
    const d = app.latest;
    if (!d) {
      continue;
    }
    if (d.status === "failed") {
      attention.push({ kind: "failed", app, deployment: d });
    } else if (d.status === "awaiting_approval") {
      attention.push({ kind: "awaiting", app, deployment: d });
    } else if (statusGroupOf(d.status) === "building") {
      attention.push({ kind: "building", app, deployment: d });
    }
  }
  const order = { failed: 0, awaiting: 1, building: 2 } as const;
  attention.sort((a, b) => order[a.kind] - order[b.kind]);

  const liveApps = data.apps.filter((a) => a.hasCurrentDeployment).length;

  const steps: SetupStep[] = [
    { id: "app", label: "Create an app", done: hasApps },
    { id: "github", label: "Connect GitHub", done: data.githubInstalled },
    { id: "deploy", label: "Ship a deployment", done: liveApps > 0 },
    { id: "keyspace", label: "Create a keyspace", done: data.keyspaces.length > 0 },
    { id: "ratelimit", label: "Add a ratelimit", done: data.ratelimits.length > 0 },
  ];

  return { shape, attention, steps, liveApps };
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
