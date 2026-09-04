"use client";

import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { trpc } from "@/lib/trpc/client";
import {
  APP_METRICS_WINDOWS,
  type AppMetricsGroup,
  type AppMetricsWindow,
  type AppResourceMetric,
} from "@unkey/clickhouse";
import type { Route } from "next";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";

export type MetricsScope = {
  appId: string;
  environmentId: string;
  window: AppMetricsWindow;
  groupBy: AppMetricsGroup;
};

const REFETCH_MS = 30_000;

function isWindow(value: string | null): value is AppMetricsWindow {
  return value !== null && (APP_METRICS_WINDOWS as readonly string[]).includes(value);
}

// URL is the single source of truth for the page state so a link to a spike
// carries the environment, window and variant with it.
export function useMetricsUrlState() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();

  const appEnvironments = useMemo(
    () => environments.filter((e) => e.appId === appId),
    [environments, appId],
  );

  const requestedEnvironmentId = searchParams.get("environmentId");
  const environmentId =
    appEnvironments.find((e) => e.id === requestedEnvironmentId)?.id ??
    appEnvironments.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id ??
    appEnvironments.at(0)?.id;

  const rangeParam = searchParams.get("range");
  const window: AppMetricsWindow = isWindow(rangeParam) ? rangeParam : "1w";
  const variant = searchParams.get("v") === "2" ? 2 : 1;
  const detail = searchParams.get("metric");
  const selectedDeployments = (searchParams.get("deployments") ?? "").split(",").filter(Boolean);

  const set = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams.toString());
      for (const [k, v] of Object.entries(patch)) {
        if (v === null) {
          next.delete(k);
        } else {
          next.set(k, v);
        }
      }
      router.replace(`${pathname}?${next.toString()}` as Route, { scroll: false });
    },
    [router, pathname, searchParams],
  );

  return {
    appId,
    environments: appEnvironments,
    isEnvironmentsLoading,
    environmentId,
    window,
    variant,
    detail,
    setEnvironmentId: (id: string) => set({ environmentId: id }),
    setWindow: (w: AppMetricsWindow) => set({ range: w }),
    setVariant: (v: number) => set({ v: String(v) }),
    setDetail: (metric: string | null) => set({ metric }),
    selectedDeployments,
    setSelectedDeployments: (ids: string[]) =>
      set({ deployments: ids.length > 0 ? ids.join(",") : null }),
  };
}

export function useResourceSeries(scope: MetricsScope, metric: AppResourceMetric) {
  return trpc.deploy.metrics.getAppResourceTimeseries.useQuery(
    { ...scope, metric },
    { refetchInterval: REFETCH_MS, keepPreviousData: true },
  );
}

export function useRequestSeries(scope: MetricsScope) {
  return trpc.deploy.metrics.getAppRequestTimeseries.useQuery(scope, {
    refetchInterval: REFETCH_MS,
    keepPreviousData: true,
  });
}

export function useLatencySeries(scope: MetricsScope) {
  return trpc.deploy.metrics.getAppLatencyTimeseries.useQuery(scope, {
    refetchInterval: REFETCH_MS,
    keepPreviousData: true,
  });
}

export function useDeploymentMarkers(scope: Omit<MetricsScope, "groupBy">) {
  return trpc.deploy.metrics.getAppDeploymentMarkers.useQuery(
    { appId: scope.appId, environmentId: scope.environmentId, window: scope.window },
    { refetchInterval: REFETCH_MS, keepPreviousData: true },
  );
}

export type DeploymentInfo = {
  id: string;
  environmentId: string;
  sha: string | null;
  message: string | null;
  branch: string;
  status: string;
  createdAt: number;
};

// Deployment metadata for series labels and breakdown rows. Series are keyed
// by deployment id; anything not in the project collection (very old or
// deleted) falls back to the raw id.
export function useDeploymentIndex(): Map<string, DeploymentInfo> {
  const { deployments } = useProjectData();
  return useMemo(() => {
    const map = new Map<string, DeploymentInfo>();
    for (const d of deployments) {
      map.set(d.id, {
        id: d.id,
        environmentId: d.environmentId,
        sha: d.gitCommitSha,
        message: d.gitCommitMessage,
        branch: d.gitBranch,
        status: d.status,
        createdAt: d.createdAt,
      });
    }
    return map;
  }, [deployments]);
}

// Deployments of one environment, newest first. The position in this list is
// also the deployment's colour, so a deployment keeps its colour across every
// chart and the picker.
export function useEnvironmentDeployments(environmentId: string): DeploymentInfo[] {
  const index = useDeploymentIndex();
  return useMemo(
    () =>
      [...index.values()]
        .filter((d) => d.environmentId === environmentId)
        .sort((a, b) => b.createdAt - a.createdAt),
    [index, environmentId],
  );
}
