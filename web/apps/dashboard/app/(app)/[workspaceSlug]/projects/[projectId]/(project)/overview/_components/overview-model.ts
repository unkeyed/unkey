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
