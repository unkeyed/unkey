"use client";

import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { Button, Empty, ResourceListBody, ResourceListContent } from "@unkey/ui";
import { IconBookBookmarkOutline18 } from "nucleo-ui-outline-18";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { useAppId, useProjectData } from "../../data-provider";
import { useDeployments } from "../hooks/use-deployments";
import { DeploymentRow } from "./deployment-row";
import { DeploymentsSkeleton } from "./deployments-skeleton";

type DeploymentsCardListProps = {
  limit?: number;
};

export function DeploymentsCardList({ limit }: DeploymentsCardListProps = {}) {
  const { deployments } = useDeployments();
  const { projectId } = useProjectData();
  const appId = useAppId();
  const appsQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appsQuery.data?.[0];
  const currentDeploymentId = app?.currentDeploymentId;
  const workspace = useWorkspaceNavigation();

  if (deployments.isLoading) {
    return <DeploymentsSkeleton />;
  }

  const data = typeof limit === "number" ? deployments.data.slice(0, limit) : deployments.data;

  if (data.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex w-full items-center justify-center px-4 py-16">
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
                  <IconBookBookmarkOutline18 />
                  Learn about Deployments
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </div>
      </ResourceListContent>
    );
  }

  return (
    <ResourceListContent>
      <ResourceListBody>
        {data.map(({ deployment, environment }) => {
          const isCurrent = currentDeploymentId === deployment.id;
          return (
            <DeploymentRow
              key={deployment.id}
              deployment={deployment}
              environment={environment}
              isCurrent={isCurrent}
              isRolledBack={isCurrent && (app?.isRolledBack ?? false)}
              href={routes.projects.apps.deployment({
                workspaceSlug: workspace.slug,
                projectId,
                appId: deployment.appId,
                deploymentId: deployment.id,
              })}
            />
          );
        })}
      </ResourceListBody>
    </ResourceListContent>
  );
}
