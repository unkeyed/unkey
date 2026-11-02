"use client";

import { type ListView, listOf, useInfiniteList } from "@/lib/loadable/list";
import {
  type Loadable,
  fromQuery,
  mapLoadable,
  readyLoadable,
  zipLoadable,
} from "@/lib/loadable/loadable";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useInfiniteQuery, useQueries, useQuery } from "@tanstack/react-query";
import type { Callers } from "../model/callers";
import { removalOf, removesPreview } from "../model/plan";
import {
  type ConnectionTarget,
  type ConnectionTargets,
  type Environment,
  type ListedConnection,
  type TargetDeployment,
  connectionTarget,
} from "../model/target";
import { mergeDeploymentTargets } from "../model/target-options";

export type RetainedConnection = {
  id: string;
  targetAppName: string;
  deployments: number;
};

type ConnectionsSnapshot = {
  connections: ListedConnection[];
  targets: ConnectionTargets;
};

export type ConnectionRowModel = {
  connection: ListedConnection;
  target: ConnectionTarget;
  alsoPreview: boolean;
};

export type CallerSnapshot = ConnectionsSnapshot & { environment: Environment };

export function useConnectionsData({
  projectId,
  appId,
  production,
  preview,
}: {
  projectId: string;
  appId: string;
  production: Environment;
  preview: Environment | null;
}) {
  const productionSnapshot = mapLoadable(
    fromQuery(useQuery(snapshotQuery(projectId, appId, production.id))),
    (snapshot): CallerSnapshot => ({ ...snapshot, environment: production }),
  );
  const previewQueries = preview ? [snapshotQuery(projectId, appId, preview.id)] : [];
  const [previewQuery] = useQueries({ queries: previewQueries });
  const previewSnapshot: Loadable<CallerSnapshot | null> =
    preview && previewQuery
      ? mapLoadable(fromQuery(previewQuery), (snapshot) => ({ ...snapshot, environment: preview }))
      : readyLoadable(null);
  const callers = zipLoadable(
    productionSnapshot,
    previewSnapshot,
    (productionCaller, previewCaller): Callers<CallerSnapshot> => ({
      production: productionCaller,
      preview: previewCaller,
    }),
  );
  return {
    callers,
    view: listOf(
      mapLoadable(callers, (loaded) =>
        loaded.production.connections.map(
          (connection): ConnectionRowModel => ({
            connection,
            target: connectionTarget(connection, loaded.production),
            alsoPreview: removesPreview(removalOf(loaded, connection.targetAppId)),
          }),
        ),
      ),
    ),
    retained: listOf(fromQuery(useRetained(projectId, appId, production.id))),
  };
}

function snapshotQuery(projectId: string, appId: string, environmentId: string) {
  return {
    queryKey: queryKeys.connections.list(projectId, appId, environmentId),
    queryFn: async ({ signal }: { signal?: AbortSignal }): Promise<ConnectionsSnapshot> => {
      const scope = { project: projectId, app: appId, environment: environmentId };
      const client = getUnkeyClient().connections;
      const [pages, targets] = await Promise.all([
        client.listConnections(scope, { signal }).then((firstPage) => Array.fromAsync(firstPage)),
        client.listTargets(scope, { signal }),
      ]);
      return {
        connections: pages
          .flatMap((page) => page.result.data)
          .sort((a, b) => a.targetAppName.localeCompare(b.targetAppName)),
        targets: targets.data,
      };
    },
  };
}

function useRetained(projectId: string, appId: string, environmentId: string) {
  return useQuery({
    queryKey: queryKeys.connections.retained(projectId, appId, environmentId),
    queryFn: async ({ signal }): Promise<RetainedConnection[]> =>
      (
        await getUnkeyClient().connections.listRetainedConnections(
          { project: projectId, app: appId, environment: environmentId },
          { signal },
        )
      ).data,
  });
}

export function useTargetDeployments({
  projectId,
  appId,
  environmentId,
  targetAppId,
  known,
  enabled,
}: {
  projectId: string;
  appId: string;
  environmentId: string;
  targetAppId: string;
  known: TargetDeployment[];
  enabled: boolean;
}): { choices: TargetDeployment[]; list: ListView<TargetDeployment> } {
  const query = useInfiniteQuery({
    queryKey: queryKeys.connections.targets(projectId, appId, environmentId, targetAppId),
    queryFn: ({ pageParam, signal }) =>
      getUnkeyClient().connections.listTargets(
        {
          project: projectId,
          app: appId,
          environment: environmentId,
          targetAppId,
          cursor: pageParam,
        },
        { signal },
      ),
    enabled,
    getNextPageParam: (page) => (page.pagination.hasMore && page.pagination.cursor) || undefined,
  });
  const list = useInfiniteList(query, (page) =>
    page.data.deployments.filter((deployment) => deployment.appId === targetAppId),
  );
  return {
    choices: mergeDeploymentTargets(
      known.filter((deployment) => deployment.appId === targetAppId),
      list.status === "ready" ? [...list.rows] : [],
    ),
    list,
  };
}
