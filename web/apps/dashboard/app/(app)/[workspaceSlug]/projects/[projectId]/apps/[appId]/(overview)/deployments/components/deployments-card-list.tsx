"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { BookBookmark } from "@unkey/icons";
import {
  Button,
  Empty,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
} from "@unkey/ui";
import { useProjectData } from "../../data-provider";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { useDeployments } from "../hooks/use-deployments";
import { useFilters } from "../hooks/use-filters";
import { DeploymentRow } from "./deployment-row";
import { DeploymentsSkeleton } from "./deployments-skeleton";

export function DeploymentsCardList() {
  const {
    rows,
    isLoading,
    isError,
    refetch,
    isFiltered,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useDeployments();
  const { updateFilters } = useFilters();
  const { projectId } = useProjectData();
  const { app, currentDeployment, isRolledBack } = useAppCurrentDeployment();
  const workspace = useWorkspaceNavigation();

  if (isLoading) {
    return <DeploymentsSkeleton />;
  }

  if (isError && rows.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex w-full items-center justify-center gap-3 px-4 py-16">
          <span role="alert" className="text-error-11 text-sm">
            We couldn't load deployments.
          </span>
          <Button size="md" variant="outline" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      </ResourceListContent>
    );
  }

  if (rows.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex w-full items-center justify-center px-4 py-16">
          {isFiltered ? (
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No deployments match these filters</Empty.Title>
              <Empty.Description className="text-left">
                Widen the environment, status, branch or time range to see more deployments.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <Button size="md" variant="outline" onClick={() => updateFilters([])}>
                  Clear filters
                </Button>
              </Empty.Actions>
            </Empty>
          ) : (
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
          )}
        </div>
      </ResourceListContent>
    );
  }

  return (
    <ResourceListContent>
      <ResourceListBody>
        {rows.map(({ deployment, environment }) => (
          <DeploymentRow
            key={deployment.id}
            deployment={deployment}
            environment={environment}
            repoFullName={app?.repositoryFullName ?? null}
            currentDeployment={currentDeployment}
            isRolledBack={isRolledBack}
            href={routes.projects.apps.deployment({
              workspaceSlug: workspace.slug,
              projectId,
              appId: deployment.appId,
              deploymentId: deployment.id,
            })}
          />
        ))}
      </ResourceListBody>
      {hasNextPage && (
        <ResourceListFooter>
          <Button
            size="md"
            variant="outline"
            disabled={isFetchingNextPage}
            loading={isFetchingNextPage}
            onClick={() => fetchNextPage()}
          >
            Load more
          </Button>
        </ResourceListFooter>
      )}
    </ResourceListContent>
  );
}
