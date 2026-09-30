"use client";

import { routes } from "@/lib/navigation/routes";
import {
  Button,
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
  ResourceListHeader,
} from "@unkey/ui";
import Link from "next/link";
import { useProjectData } from "../../../../data-provider";
import { useAppCurrentDeployment } from "../../../../hooks/use-app-current-deployment";
import { DeploymentsSkeleton } from "../../../deployments/components/deployments-skeleton";
import { useAppScope } from "../../../environment-context";
import { ActiveBranchRow } from "./active-branch-row";
import { useActiveBranches } from "./use-active-branches";

export function ActiveBranches() {
  const scope = useAppScope();
  const { environments } = useProjectData();
  const { branches, isLoading, isError, refetch, hasNextPage, isFetchingNextPage, fetchNextPage } =
    useActiveBranches();
  const {
    app,
    currentDeployment,
    isRolledBack,
    isLoading: isAppLoading,
  } = useAppCurrentDeployment();

  const loading = isLoading || isAppLoading;
  const repoFullName = app?.repositoryFullName ?? null;
  if (!loading && !isError && repoFullName === null && branches.length === 0) {
    return null;
  }

  const environmentById = new Map(environments.map((e) => [e.id, e]));

  return (
    <ResourceList>
      <ResourceListHeader className="flex-row items-center justify-between">
        <h2 className="font-medium text-gray-12 text-sm">Active Branches</h2>
        <Link
          href={routes.projects.apps.deployments(scope)}
          className="text-sm text-gray-11 transition-colors hover:text-gray-12"
        >
          View all deployments
        </Link>
      </ResourceListHeader>
      {loading ? (
        <DeploymentsSkeleton rows={4} />
      ) : (
        <ResourceListContent>
          {isError ? (
            <div className="flex items-center justify-center gap-3 px-4 py-10">
              <span role="alert" className="text-error-11 text-sm">
                We couldn't load active branches.
              </span>
              <Button size="md" variant="outline" onClick={() => refetch()}>
                Retry
              </Button>
            </div>
          ) : branches.length === 0 ? (
            <div className="px-4 py-10 text-center text-sm text-gray-9">
              No branch deployments yet.
            </div>
          ) : (
            <ResourceListBody>
              {branches.map((deployment) => (
                <ActiveBranchRow
                  key={deployment.id}
                  branch={deployment.gitBranch}
                  deployment={deployment}
                  environment={environmentById.get(deployment.environmentId)}
                  repoFullName={repoFullName}
                  currentDeployment={currentDeployment}
                  isRolledBack={isRolledBack}
                  href={routes.projects.apps.deployment({ ...scope, deploymentId: deployment.id })}
                />
              ))}
            </ResourceListBody>
          )}
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
      )}
    </ResourceList>
  );
}
