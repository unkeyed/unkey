"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { findRolledBackFrom } from "@/lib/collections/deploy/rollback";
import {
  RECENT_DEPLOYMENTS_LIMIT,
  useApp,
  useAppScope,
  useCurrentDeployment,
  useEnvironments,
  useProductionDeployments,
  useRecentDeployments,
} from "@/lib/deploy-data";
import { routes } from "@/lib/navigation/routes";
import { ResourceList, ResourceListBody, ResourceListContent, ResourceListHeader } from "@unkey/ui";
import Link from "next/link";
import { DeploymentRow } from "../../deployments/components/deployment-row";
import { DeploymentsSkeleton } from "../../deployments/components/deployments-skeleton";

export function RecentDeployments() {
  const workspace = useWorkspaceNavigation();
  const { projectId, appId } = useAppScope();
  const { data: environments } = useEnvironments();
  const { data: deployments, isLoading: isDeploymentsLoading } = useRecentDeployments({
    limit: RECENT_DEPLOYMENTS_LIMIT,
  });
  const { data: app } = useApp();
  const currentDeployment = useCurrentDeployment().data ?? undefined;
  const isRolledBack = app?.isRolledBack ?? false;
  const { data: productionDeployments } = useProductionDeployments({ enabled: isRolledBack });
  const rolledBackFromId =
    isRolledBack && currentDeployment
      ? findRolledBackFrom(productionDeployments, currentDeployment)?.id
      : undefined;

  return (
    <ResourceList>
      <ResourceListHeader className="flex-row items-center justify-between">
        <h2 className="font-medium text-gray-12 text-sm">Recent Deployments</h2>
        <Link
          href={routes.projects.apps.deployments({
            workspaceSlug: workspace.slug,
            projectId,
            appId,
          })}
          className="text-sm text-gray-11 transition-colors hover:text-gray-12"
        >
          View all deployments
        </Link>
      </ResourceListHeader>
      {isDeploymentsLoading ? (
        <DeploymentsSkeleton rows={5} />
      ) : (
        <ResourceListContent>
          <ResourceListBody>
            {deployments.slice(0, RECENT_DEPLOYMENTS_LIMIT).map((deployment) => (
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
