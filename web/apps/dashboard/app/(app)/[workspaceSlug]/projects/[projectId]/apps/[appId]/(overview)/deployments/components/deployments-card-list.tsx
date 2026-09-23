"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { findRolledBackFrom } from "@/lib/collections/deploy/rollback";
import { routes } from "@/lib/navigation/routes";
import { IconBookBookmarkOutline18, IconSquareBulletListOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
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

  const rolledBackFromId =
    isRolledBack && currentDeployment
      ? findRolledBackFrom(
          rows.map((row) => row.deployment),
          currentDeployment,
        )?.id
      : undefined;

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
        {isFiltered ? (
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconSquareBulletListOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No deployments match these filters</EmptyStateTitle>
              <EmptyStateDescription>
                Widen the environment, status, branch or time range to see more deployments.
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <Button size="md" variant="outline" onClick={() => updateFilters([])}>
                Clear filters
              </Button>
            </EmptyStateActions>
          </EmptyState>
        ) : (
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconSquareBulletListOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No Active Deployments</EmptyStateTitle>
              <EmptyStateDescription>
                {app?.sourceType === "oci"
                  ? "Deploy the configured image or enter another image reference to get started."
                  : "Push to your connected repository or trigger a manual deployment to get started."}{" "}
                Cancelled, skipped, superseded, and stopped deployments are hidden by default.
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <a
                href="https://www.unkey.com/docs/build-and-deploy/deployments"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="outline" size="md">
                  <IconBookBookmarkOutline18 />
                  Learn about Deployments
                </Button>
              </a>
            </EmptyStateActions>
          </EmptyState>
        )}
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
            isRolledBackFrom={deployment.id === rolledBackFromId}
            liveSince={app?.updatedAt ?? null}
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
