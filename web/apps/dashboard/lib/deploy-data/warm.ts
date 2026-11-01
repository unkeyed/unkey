"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { warmQueries } from "@/lib/collections/warm-queries";
import { createLiveQueryCollection } from "@tanstack/react-db";
import {
  PRODUCTION_LIMIT,
  RECENT_DEPLOYMENTS_LIMIT,
  deploymentByIdQueryFor,
  recentDeploymentsQueryFor,
  warmDeploymentsPage,
} from "./deployments";
import { customDomainsQueryFor, domainsQueryFor, environmentsQueryFor } from "./queries";

// Loads what the app overview reads first, so a click after hovering finds it held
export function warmAppPage(projectId: string, appId: string): void {
  warmDeploymentsPage(projectId, appId);
  const currentDeploymentId = collection.apps.get(appId)?.currentDeploymentId;
  const productionId = [...collection.environments.values()].find(
    (e) => e.appId === appId && e.kind === ENVIRONMENT_KIND.production,
  )?.id;
  warmQueries(`app/${projectId}/${appId}`, () => [
    createLiveQueryCollection(
      recentDeploymentsQueryFor(projectId, appId, null, RECENT_DEPLOYMENTS_LIMIT),
    ),
    createLiveQueryCollection(domainsQueryFor(projectId, appId)),
    createLiveQueryCollection(customDomainsQueryFor(projectId, appId)),
    createLiveQueryCollection(environmentsQueryFor(projectId, appId)),
    ...(productionId
      ? [
          createLiveQueryCollection(
            recentDeploymentsQueryFor(projectId, appId, productionId, PRODUCTION_LIMIT),
          ),
        ]
      : []),
    ...(currentDeploymentId
      ? [createLiveQueryCollection(deploymentByIdQueryFor(projectId, appId, currentDeploymentId))]
      : []),
  ]);
}
