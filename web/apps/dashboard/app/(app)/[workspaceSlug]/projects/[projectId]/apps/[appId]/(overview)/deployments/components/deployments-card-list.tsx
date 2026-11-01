"use client";

import { ListPager } from "@/components/list-pager";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { findRolledBackFrom } from "@/lib/collections/deploy/rollback";
import {
  type DeploymentListFilter,
  useApp,
  useAppScope,
  useCurrentDeployment,
  useDeploymentsPage,
  useEnvironments,
} from "@/lib/deploy-data";
import { moreFooterOf } from "@/lib/loadable/list";
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
} from "@unkey/ui";
import { useMemo } from "react";
import { buildDeploymentListInput } from "../hooks/deployment-list-input";
import { useFilters } from "../hooks/use-filters";
import { DeploymentRow } from "./deployment-row";
import { DeploymentsSkeleton } from "./deployments-skeleton";

export function DeploymentsCardList() {
  const { filters, isFiltered, updateFilters } = useFilters();
  const { data: environments, isLoading: isEnvironmentsLoading } = useEnvironments();
  const { filter, cannotMatch } = useMemo(
    () => buildDeploymentListInput(filters, environments),
    [filters, environments],
  );

  if (isEnvironmentsLoading) {
    return <DeploymentsSkeleton />;
  }
  if (cannotMatch) {
    return (
      <ResourceListContent>
        <NoMatchingDeployments onClear={() => updateFilters([])} />
      </ResourceListContent>
    );
  }
  return (
    <DeploymentsCardListResults
      filter={filter}
      isFiltered={isFiltered}
      onClearFilters={() => updateFilters([])}
    />
  );
}

function DeploymentsCardListResults({
  filter,
  isFiltered,
  onClearFilters,
}: {
  filter: DeploymentListFilter;
  isFiltered: boolean;
  onClearFilters: () => void;
}) {
  const { rows, isLoading, isError, hasNextPage, isFetchingNextPage, fetchNextPage, refetch } =
    useDeploymentsPage(filter);
  const { projectId } = useAppScope();
  const { data: environments } = useEnvironments();
  const { data: app } = useApp();
  const currentDeployment = useCurrentDeployment().data ?? undefined;
  const isRolledBack = app?.isRolledBack ?? false;
  const workspace = useWorkspaceNavigation();

  const rolledBackFromId =
    isRolledBack && currentDeployment ? findRolledBackFrom(rows, currentDeployment)?.id : undefined;

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
          <NoMatchingDeployments onClear={onClearFilters} />
        ) : (
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconSquareBulletListOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No Active Deployments</EmptyStateTitle>
              <EmptyStateDescription>
                {app?.sourceType === "oci"
                  ? "Deploy the configured image or enter another image reference."
                  : "Push to your connected repository or trigger a manual deployment."}
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <a
                href="https://www.unkey.com/docs/build-and-deploy/deployments"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="outline" size="sm">
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
        {rows.map((deployment) => (
          <DeploymentRow
            key={deployment.id}
            deployment={deployment}
            environment={environments.find((e) => e.id === deployment.environmentId)}
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
      <ListPager
        state={moreFooterOf({
          hasMore: hasNextPage,
          isLoadingMore: isFetchingNextPage,
          loadMore: () => fetchNextPage(),
        })}
        noun="deployments"
      />
    </ResourceListContent>
  );
}

function NoMatchingDeployments({ onClear }: { onClear: () => void }) {
  return (
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
        <Button size="sm" variant="outline" onClick={onClear}>
          Clear filters
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}
