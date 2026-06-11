"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useProjectData } from "../../data-provider";
import { useDeployments } from "../hooks/use-deployments";
import { DeploymentRow } from "./deployment-row";
import { DeploymentsSkeleton } from "./deployments-skeleton";

export function DeploymentsCardList() {
  const { deployments } = useDeployments();
  const { project, projectId } = useProjectData();
  const currentDeploymentId = project?.currentDeploymentId;
  const workspace = useWorkspaceNavigation();

  if (deployments.isLoading) {
    return <DeploymentsSkeleton />;
  }

  if (deployments.data.length === 0) {
    return (
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
        <div className="w-full flex justify-center items-center py-16 px-4">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No Deployments Found</Empty.Title>
            <Empty.Description className="text-left">
              There are no deployments yet. Push to your connected repository or trigger a manual
              deployment to get started.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/build-and-deploy/deployments"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button size="md">
                  <BookBookmark />
                  Learn about Deployments
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {deployments.data.map(({ deployment, environment }) => {
        const isCurrent = currentDeploymentId === deployment.id;
        return (
          <DeploymentRow
            key={deployment.id}
            deployment={deployment}
            environment={environment}
            isCurrent={isCurrent}
            isRolledBack={isCurrent && (project?.isRolledBack ?? false)}
            href={routes.projects.apps.deployment({
              workspaceSlug: workspace.slug,
              projectId,
              appId: deployment.appId,
              deploymentId: deployment.id,
            })}
          />
        );
      })}
    </div>
  );
}
