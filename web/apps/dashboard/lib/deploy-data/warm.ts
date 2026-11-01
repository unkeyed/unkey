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
import {
  customDomainsQueryFor,
  domainsQueryFor,
  environmentsQueryFor,
  projectAppsQueryFor,
} from "./queries";

// Loads a project's apps, so opening the project after hovering it finds them loaded
export function warmProjectPage(projectId: string): void {
  warmQueries(`project/${projectId}`, () => [
    createLiveQueryCollection(projectAppsQueryFor(projectId)),
  ]);
}

// Loads what the app overview reads first, so a click after hovering finds it loaded
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

// Loads one deployment by id, so opening its page after hovering its row finds it loaded
export function warmDeployment(projectId: string, appId: string, deploymentId: string): void {
  warmQueries(`deployment/${deploymentId}`, () => [
    createLiveQueryCollection(deploymentByIdQueryFor(projectId, appId, deploymentId)),
  ]);
}
