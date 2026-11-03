"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment } from "@/lib/deploy-data";
import { useConnectionsEnabled } from "@/lib/flags/use-connections-enabled";
import { type ListView, listOf } from "@/lib/loadable/list";
import { type Loadable, fromQuery, mapLoadable } from "@/lib/loadable/loadable";
import { routes } from "@/lib/navigation/routes";
import { queryKeys } from "@/lib/query-keys";
import { trpcClient } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { useQuery } from "@tanstack/react-query";
import type { inferRouterOutputs } from "@trpc/server";
import type { Route } from "next";

type DeploymentOutputs = inferRouterOutputs<Router>["deploy"]["deployment"];
export type DeploymentConnectionSummary =
  DeploymentOutputs["listConnections"]["connections"][number];
type DeploymentConnectionPin = DeploymentOutputs["listConnectionPins"]["pins"][number];

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
  deployment: Pick<Deployment, "id" | "projectId">,
): TruncatedList<DeploymentConnectionSummary> | null {
  const enabled = useConnectionsEnabled();
  const query = useQuery({
    enabled,
    queryKey: queryKeys.connections.deployment(deployment.id),
    queryFn: ({ signal }) =>
      trpcClient.deploy.deployment.listConnections.query(
        { projectId: deployment.projectId, deploymentId: deployment.id },
        { signal },
      ),
  });
  if (!enabled) {
    return null;
  }
  return truncatedListOf(fromQuery(query), ({ connections }) => connections);
}

export function usePinningApps(deployment: Deployment): TruncatedList<PinningApp> | null {
  const enabled = useConnectionsEnabled();
  const workspace = useWorkspaceNavigation();
  const query = useQuery({
    enabled,
    queryKey: queryKeys.connections.pins(deployment.id),
    queryFn: ({ signal }) =>
      trpcClient.deploy.deployment.listConnectionPins.query(
        { projectId: deployment.projectId, deploymentId: deployment.id },
        { signal },
      ),
  });
  const appHref = (appId: string) =>
    routes.projects.apps.connections({
      workspaceSlug: workspace.slug,
      projectId: deployment.projectId,
      appId,
    });
  if (!enabled) {
    return null;
  }
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
