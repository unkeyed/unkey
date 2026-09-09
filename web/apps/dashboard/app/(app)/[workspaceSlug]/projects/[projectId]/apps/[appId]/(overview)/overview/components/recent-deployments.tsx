"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { findRolledBackFrom } from "@/lib/collections/deploy/rollback";
import { routes } from "@/lib/navigation/routes";
import { ResourceList, ResourceListBody, ResourceListContent, ResourceListHeader } from "@unkey/ui";
import Link from "next/link";
import { useAppId, useProjectData } from "../../data-provider";
import { DeploymentRow } from "../../deployments/components/deployment-row";
import { DeploymentsSkeleton } from "../../deployments/components/deployments-skeleton";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";

export function RecentDeployments() {
  const workspace = useWorkspaceNavigation();
  const appId = useAppId();
  const { projectId, deployments, environments, isDeploymentsLoading } = useProjectData();
  const { app, currentDeployment, isRolledBack } = useAppCurrentDeployment();
  const rolledBackFromId =
    isRolledBack && currentDeployment
      ? findRolledBackFrom(deployments, currentDeployment)?.id
      : undefined;

  return (
    <ResourceList>
      <ResourceListHeader className="flex-row items-center justify-between">
        <h2 className="font-medium text-accent-12 text-sm">Recent Deployments</h2>
        <Link
          href={routes.projects.apps.deployments({
            workspaceSlug: workspace.slug,
            projectId,
            appId,
          })}
          className="text-[13px] text-gray-11 transition-colors hover:text-gray-12"
        >
          View all deployments
        </Link>
      </ResourceListHeader>
      {isDeploymentsLoading ? (
        <DeploymentsSkeleton rows={5} />
      ) : (
        <ResourceListContent>
          <ResourceListBody>
            {deployments.slice(0, 5).map((deployment) => (
              <DeploymentRow
                key={deployment.id}
                deployment={deployment}
                environment={environments.find((env) => env.id === deployment.environmentId)}
                repoFullName={app?.repositoryFullName ?? null}
                currentDeployment={currentDeployment}
                isRolledBack={isRolledBack}
                isRolledBackFrom={deployment.id === rolledBackFromId}
                liveSince={app?.updatedAt ?? null}
                href={routes.projects.apps.deployment({
                  workspaceSlug: workspace.slug,
                  projectId,
                  appId,
                  deploymentId: deployment.id,
                })}
              />
            ))}
          </ResourceListBody>
        </ResourceListContent>
      )}
    </ResourceList>
  );
}
