"use client";

import { ENVIRONMENT_KIND, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import { type ConnectionTargets, mergeDeploymentTargets } from "../connection-rules";
import { type ConnectionEnvs, type MergedConnection, mergeConnections } from "../model/merge";

export type ConnectionsView =
  | { type: "error" }
  | { type: "loading" }
  | { type: "empty" }
  | { type: "list" };

export type RetainedConnection = {
  id: string;
  env: EnvironmentKind;
  targetAppName: string;
  deployments: number;
};

export type RetainedView =
  | { type: "error" }
  | { type: "loading" }
  | { type: "ready"; rows: RetainedConnection[] };

export type ConnectionsData = {
  projectId: string;
  appId: string;
  envs: ConnectionEnvs;
  merged: MergedConnection[];
  targets: ConnectionTargets;
  retained: RetainedView;
  requiresExplicitPreviewTarget: boolean;
  view: ConnectionsView;
  retry: () => void;
  canWrite: boolean;
};

const EMPTY_TARGETS: ConnectionTargets = { apps: [], environments: [], deployments: [] };

export function useConnectionsData(): ConnectionsData {
  const { projectId, apps, environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();
  const productionEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.production) ?? null;
  const previewEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.preview) ?? null;
  const envs = useMemo(
    () => ({ production: productionEnv, preview: previewEnv }),
    [productionEnv, previewEnv],
  );

  const productionList = useConnectionList(projectId, appId, productionEnv?.id);
  const previewList = useConnectionList(projectId, appId, previewEnv?.id);
  const productionTargets = useCallerTargets(projectId, appId, productionEnv?.id);
  const previewTargets = useCallerTargets(projectId, appId, previewEnv?.id);
  const productionRetained = useRetained(projectId, appId, productionEnv?.id);
  const previewRetained = useRetained(projectId, appId, previewEnv?.id);

  const productionRows = productionList.data;
  const previewRows = previewList.data;
  const merged = useMemo(
    () => mergeConnections([...(productionRows ?? []), ...(previewRows ?? [])], envs),
    [productionRows, previewRows, envs],
  );

  const productionData = productionTargets.data;
  const previewData = previewTargets.data;
  const targets = useMemo((): ConnectionTargets => {
    const base = productionData ?? previewData;
    if (!base) {
      return EMPTY_TARGETS;
    }
    return {
      apps: base.apps,
      environments: base.environments,
      deployments: mergeDeploymentTargets(
        productionData?.deployments ?? [],
        previewData?.deployments ?? [],
      ),
    };
  }, [productionData, previewData]);

  const isLoading =
    isEnvironmentsLoading ||
    productionList.isInitialLoading ||
    previewList.isInitialLoading ||
    productionTargets.isInitialLoading ||
    previewTargets.isInitialLoading;
  const isError =
    productionList.isError ||
    previewList.isError ||
    productionTargets.isError ||
    previewTargets.isError;

  return {
    projectId,
    appId,
    envs,
    merged,
    targets,
    retained: retainedView(productionRetained, previewRetained),
    requiresExplicitPreviewTarget: apps.find((app) => app.id === appId)?.sourceType === "oci",
    view: connectionsView(isError, isLoading, merged.length),
    retry: () => {
      void productionList.refetch();
      void previewList.refetch();
      void productionTargets.refetch();
      void previewTargets.refetch();
    },
    canWrite: !isLoading && !isError && (productionEnv !== null || previewEnv !== null),
  };
}

function useConnectionList(projectId: string, appId: string, environmentId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.connections.list(projectId, appId, environmentId ?? ""),
    queryFn: async ({ signal }) => {
      const firstPage = await getUnkeyClient().connections.listConnections(
        { project: projectId, app: appId, environment: environmentId ?? "" },
        { signal },
      );
      const pages = await Array.fromAsync(firstPage);
      return pages.flatMap((page) => page.result.data);
    },
    enabled: environmentId !== undefined,
  });
}

function useCallerTargets(projectId: string, appId: string, environmentId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.connections.targets(projectId, appId, environmentId ?? ""),
    queryFn: async ({ signal }) =>
      (
        await getUnkeyClient().connections.listTargets(
          { project: projectId, app: appId, environment: environmentId ?? "" },
          { signal },
        )
      ).data,
    enabled: environmentId !== undefined,
  });
}

function useRetained(projectId: string, appId: string, environmentId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.connections.retained(projectId, appId, environmentId ?? ""),
    queryFn: async ({ signal }) =>
      (
        await getUnkeyClient().connections.listRetainedConnections(
          { project: projectId, app: appId, environment: environmentId ?? "" },
          { signal },
        )
      ).data,
    enabled: environmentId !== undefined,
  });
}

type RetainedQuery = ReturnType<typeof useRetained>;

function retainedView(production: RetainedQuery, preview: RetainedQuery): RetainedView {
  if (production.isError || preview.isError) {
    return { type: "error" };
  }
  if (production.isInitialLoading || preview.isInitialLoading) {
    return { type: "loading" };
  }
  return {
    type: "ready",
    rows: [
      ...(production.data ?? []).map((row) => ({ ...row, env: "production" as const })),
      ...(preview.data ?? []).map((row) => ({ ...row, env: "preview" as const })),
    ],
  };
}

function connectionsView(isError: boolean, isLoading: boolean, count: number): ConnectionsView {
  if (isError) {
    return { type: "error" };
  }
  if (isLoading) {
    return { type: "loading" };
  }
  return count === 0 ? { type: "empty" } : { type: "list" };
}

export type TargetDeployments = {
  deployments: ConnectionTargets["deployments"];
  hasMore: boolean;
  loadingMore: boolean;
  loadMore: () => void;
};

export function useTargetDeployments({
  projectId,
  appId,
  envs,
  targetAppId,
}: {
  projectId: string;
  appId: string;
  envs: ConnectionEnvs;
  targetAppId: string;
}): TargetDeployments {
  const environmentId = envs.production?.id ?? envs.preview?.id;
  const query = useInfiniteQuery({
    queryKey: queryKeys.connections.targets(projectId, appId, environmentId ?? "", targetAppId),
    queryFn: ({ pageParam, signal }) =>
      getUnkeyClient().connections.listTargets(
        {
          project: projectId,
          app: appId,
          environment: environmentId ?? "",
          targetAppId,
          cursor: pageParam,
        },
        { signal },
      ),
    enabled: environmentId !== undefined && targetAppId !== "",
    getNextPageParam: (page) => (page.pagination.hasMore && page.pagination.cursor) || undefined,
  });
  const pages = query.data?.pages;
  const deployments = useMemo(
    () =>
      mergeDeploymentTargets(
        [],
        (pages ?? []).flatMap((page) =>
          page.data.deployments.filter((deployment) => deployment.appId === targetAppId),
        ),
      ),
    [pages, targetAppId],
  );
  return {
    deployments,
    hasMore: !!query.hasNextPage,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void query.fetchNextPage(),
  };
}
