"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { type ListView, listOf } from "@/lib/loadable/list";
import { type Loadable, fromQuery, mapLoadable } from "@/lib/loadable/loadable";
import { routes } from "@/lib/navigation/routes";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import type {
  DeploymentConnectionPin,
  DeploymentConnectionSummary,
} from "@unkey/api/models/components";
import type { Route } from "next";

export type PinningApp = { id: string; name: string; href: Route };

export type TruncatedList<Row> = { view: ListView<Row>; truncated: boolean };

function truncatedListOf<Data extends { truncated: boolean }, Row>(
  loadable: Loadable<Data>,
  rowsOf: (data: Data) => readonly Row[],
): TruncatedList<Row> {
  return {
    view: listOf(mapLoadable(loadable, rowsOf)),
    truncated: loadable.status === "ready" && loadable.data.truncated,
  };
}

export function useDeploymentConnections(
  deploymentId: string,
): TruncatedList<DeploymentConnectionSummary> {
  const query = useQuery({
    queryKey: queryKeys.deployments.connections(deploymentId),
    queryFn: async ({ signal }) =>
      (await getUnkeyClient().deployments.listConnections({ deploymentId }, { signal })).data,
  });
  return truncatedListOf(fromQuery(query), ({ connections }) => connections);
}

export function usePinningApps(deployment: Deployment): TruncatedList<PinningApp> {
  const workspace = useWorkspaceNavigation();
  const query = useQuery({
    queryKey: queryKeys.deployments.connectionPins(deployment.id),
    queryFn: async ({ signal }) =>
      (
        await getUnkeyClient().deployments.listConnectionPins(
          { deploymentId: deployment.id },
          { signal },
        )
      ).data,
  });
  const appHref = (appId: string) =>
    routes.projects.apps.connections({
      workspaceSlug: workspace.slug,
      projectId: deployment.projectId,
      appId,
    });
  return truncatedListOf(fromQuery(query), ({ pins }) => pinningAppsOf(pins, appHref));
}

function pinningAppsOf(
  pins: readonly DeploymentConnectionPin[],
  appHref: (appId: string) => Route,
): PinningApp[] {
  const apps = new Map<string, PinningApp>();
  for (const pin of pins) {
    if (!apps.has(pin.callerAppId)) {
      apps.set(pin.callerAppId, {
        id: pin.callerAppId,
        name: pin.callerAppName,
        href: appHref(pin.callerAppId),
      });
    }
  }
  return [...apps.values()];
}
