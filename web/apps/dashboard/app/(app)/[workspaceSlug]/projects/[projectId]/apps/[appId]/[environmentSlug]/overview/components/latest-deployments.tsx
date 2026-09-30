"use client";

import { findRolledBackFrom } from "@/lib/collections/deploy/rollback";
import { routes } from "@/lib/navigation/routes";
import { ResourceList, ResourceListBody, ResourceListContent, ResourceListHeader } from "@unkey/ui";
import Link from "next/link";
import { useProjectData } from "../../../data-provider";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { DeploymentRow } from "../../deployments/components/deployment-row";
import { DeploymentsSkeleton } from "../../deployments/components/deployments-skeleton";
import { useAppEnvironment, useAppScope } from "../../environment-context";

export function LatestDeployments() {
  const scope = useAppScope();
  const { environment } = useAppEnvironment();
  const { deployments, environments, isDeploymentsLoading } = useProjectData();
  const recent = deployments.filter((d) => d.environmentId === environment.id).slice(0, 5);
  const { app, currentDeployment, isRolledBack } = useAppCurrentDeployment();
  const rolledBackFromId =
    isRolledBack && currentDeployment
      ? findRolledBackFrom(deployments, currentDeployment)?.id
      : undefined;

  return (
    <ResourceList>
      <ResourceListHeader className="flex-row items-center justify-between">
        <h2 className="font-medium text-gray-12 text-sm">Latest deployments</h2>
        <Link
          href={routes.projects.apps.deployments(scope)}
          className="text-sm text-gray-11 transition-colors hover:text-gray-12"
        >
          View all deployments
        </Link>
      </ResourceListHeader>
      {isDeploymentsLoading ? (
        <DeploymentsSkeleton rows={5} />
      ) : (
        <ResourceListContent>
          {recent.length === 0 ? (
            <div className="px-4 py-10 text-center text-sm text-gray-9">No deployments yet.</div>
          ) : (
            <ResourceListBody>
              {recent.map((deployment) => (
                <DeploymentRow
                  key={deployment.id}
                  deployment={deployment}
                  environment={environments.find((env) => env.id === deployment.environmentId)}
                  repoFullName={app?.repositoryFullName ?? null}
                  currentDeployment={currentDeployment}
                  isRolledBack={isRolledBack}
                  isRolledBackFrom={deployment.id === rolledBackFromId}
                  liveSince={app?.updatedAt ?? null}
                  href={routes.projects.apps.deployment({ ...scope, deploymentId: deployment.id })}
                />
              ))}
            </ResourceListBody>
          )}
        </ResourceListContent>
      )}
    </ResourceList>
  );
}
