"use client";

import { type ListView, listOf, useInfiniteList } from "@/lib/loadable/list";
import { fromQuery, mapLoadable } from "@/lib/loadable/loadable";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  type ConnectionTarget,
  type ConnectionTargets,
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

export type ConnectionsSnapshot = {
  connections: ListedConnection[];
  targets: ConnectionTargets;
};

export type ConnectionRowModel = { connection: ListedConnection; target: ConnectionTarget };

export function useConnectionsData(projectId: string, appId: string, environmentId: string) {
  const snapshot = fromQuery(useSnapshot(projectId, appId, environmentId));
  return {
    snapshot,
    view: listOf(
      mapLoadable(snapshot, ({ connections, targets }) =>
        connections.map(
          (connection): ConnectionRowModel => ({
            connection,
            target: connectionTarget(connection, targets),
          }),
        ),
      ),
    ),
    retained: listOf(fromQuery(useRetained(projectId, appId, environmentId))),
  };
}

function useSnapshot(projectId: string, appId: string, environmentId: string) {
  return useQuery({
    queryKey: queryKeys.connections.list(projectId, appId, environmentId),
    queryFn: async ({ signal }): Promise<ConnectionsSnapshot> => {
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
  });
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
    enabled: enabled && targetAppId !== "",
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
