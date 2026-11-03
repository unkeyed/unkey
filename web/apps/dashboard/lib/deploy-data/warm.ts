"use client";

import { warmQueries } from "@/lib/collections/warm-queries";
import { getApp } from "@/lib/queries/apps";
import {
  ENVIRONMENT_KIND,
  type Environment,
  appEnvironmentsOptions,
} from "@/lib/queries/environments";
import { queryKeys } from "@/lib/query-keys";
import { createLiveQueryCollection } from "@tanstack/react-db";
import type { QueryClient } from "@tanstack/react-query";
import {
  PRODUCTION_LIMIT,
  RECENT_DEPLOYMENTS_LIMIT,
  deploymentDetailOptions,
  recentDeploymentsOptions,
  warmDeploymentsPage,
} from "./deployments";
import { customDomainsQueryFor, domainsQueryFor, policiesQueryFor } from "./queries";

// Loads what the app overview reads first, so a click after hovering finds it loaded
export function warmAppPage(queryClient: QueryClient, projectId: string, appId: string): void {
  warmDeploymentsPage(queryClient, projectId, appId);
  void queryClient.prefetchQuery(
    recentDeploymentsOptions(queryClient, projectId, appId, null, RECENT_DEPLOYMENTS_LIMIT),
  );
  const currentDeploymentId = getApp(queryClient, projectId, appId)?.currentDeploymentId;
  if (currentDeploymentId) {
    warmDeployment(queryClient, projectId, currentDeploymentId);
  }
  warmQueries(`app/${projectId}/${appId}`, () => {
    // The production id is only known once the environments arrive
    queryClient
      .prefetchQuery(appEnvironmentsOptions(projectId, appId))
      .then(() => warmProduction(queryClient, projectId, appId))
      .catch(() => {});
    return [
      createLiveQueryCollection(domainsQueryFor(projectId, appId)),
      createLiveQueryCollection(customDomainsQueryFor(projectId, appId)),
    ];
  });
}

// Loads one deployment by id, so opening its page after hovering its row finds it loaded
export function warmDeployment(
  queryClient: QueryClient,
  projectId: string,
  deploymentId: string,
): void {
  void queryClient.prefetchQuery(deploymentDetailOptions(queryClient, projectId, deploymentId));
}

function warmProduction(queryClient: QueryClient, projectId: string, appId: string): void {
  const productionId = queryClient
    .getQueryData<Environment[]>(queryKeys.environments.app(projectId, appId))
    ?.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id;
  if (!productionId) {
    return;
  }
  void queryClient.prefetchQuery(
    recentDeploymentsOptions(queryClient, projectId, appId, productionId, PRODUCTION_LIMIT),
  );
  warmQueries(`app/${projectId}/${appId}/production`, () => [
    createLiveQueryCollection(policiesQueryFor(projectId, appId, productionId)),
  ]);
}
