"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import type { ReactNode } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import { useDeployments } from "../hooks/use-deployments";
import { DeploymentRow } from "./deployment-row";
import { DeploymentsSkeleton } from "./deployments-skeleton";

function ListHeader({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="px-4 py-3 border-b border-grayA-4 flex items-center justify-between gap-2">
      <h2 className="text-sm font-medium text-accent-12">{title}</h2>
      {action}
    </div>
  );
}

type DeploymentsCardListProps = {
  limit?: number;
  title?: string;
  headerAction?: ReactNode;
};

export function DeploymentsCardList({ limit, title, headerAction }: DeploymentsCardListProps = {}) {
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
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
        {title && <ListHeader title={title} action={headerAction} />}
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
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      {title && <ListHeader title={title} action={headerAction} />}
      <div className="divide-y divide-grayA-4">
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
      </div>
    </div>
  );
}
