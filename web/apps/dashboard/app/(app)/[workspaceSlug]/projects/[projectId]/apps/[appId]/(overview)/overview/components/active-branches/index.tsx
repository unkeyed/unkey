"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useApp, useAppScope, useCurrentDeployment, useEnvironments } from "@/lib/deploy-data";
import { routes } from "@/lib/navigation/routes";
import { match } from "@unkey/match";
import { ResourceList, ResourceListBody, ResourceListContent, ResourceListHeader } from "@unkey/ui";
import Link from "next/link";
import { DeploymentsSkeleton } from "../../../deployments/components/deployments-skeleton";
import { ActiveBranchRow } from "./active-branch-row";
import { useActiveBranches } from "./use-active-branches";

export function ActiveBranches() {
  const workspace = useWorkspaceNavigation();
  const { projectId, appId } = useAppScope();
  const { data: environments } = useEnvironments();
  const view = useActiveBranches();
  const { data: app } = useApp();
  const current = useCurrentDeployment();
  const currentDeployment = current.data ?? undefined;
  const isRolledBack = app?.isRolledBack ?? false;
  const isAppLoading = current.isLoading;

  const repoFullName = app?.repositoryFullName ?? null;
  if (app?.sourceType === "oci") {
    return null;
  }
  if (!isAppLoading && view.status === "empty" && repoFullName === null) {
    return null;
  }

  const environmentById = new Map(environments.map((e) => [e.id, e]));

  return (
    <ResourceList>
      <ResourceListHeader className="flex-row items-center justify-between">
        <h2 className="font-medium text-gray-12 text-sm">Active Branches</h2>
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
      {isAppLoading ? (
        <DeploymentsSkeleton rows={4} />
      ) : (
        match(view)
          .with({ status: "loading" }, () => <DeploymentsSkeleton rows={4} />)
          .with({ status: "failed" }, ({ error, retry }) => (
            <ResourceListContent>
              <LoadError
                title="Could not load active branches"
                error={error}
                frame="none"
                onRetry={retry}
              />
            </ResourceListContent>
          ))
          .with({ status: "empty" }, () => (
            <ResourceListContent>
              <div className="px-4 py-10 text-center text-sm text-gray-9">
                No branch deployments yet.
              </div>
            </ResourceListContent>
          ))
          .with({ status: "ready" }, ({ rows, footer }) => (
            <ResourceListContent>
              <ResourceListBody>
                {rows.map((deployment) => (
                  <ActiveBranchRow
                    key={deployment.id}
                    branch={deployment.gitBranch}
                    deployment={deployment}
                    environment={environmentById.get(deployment.environmentId)}
                    repoFullName={repoFullName}
                    currentDeployment={currentDeployment}
                    isRolledBack={isRolledBack}
                    href={routes.projects.apps.deployment({
                      workspaceSlug: workspace.slug,
                      projectId,
                      appId,
                      deploymentId: deployment.id,
                    })}
                  />
                ))}
              </ResourceListBody>
              <ListPager state={footer} noun="active branches" />
            </ResourceListContent>
          ))
          .exhaustive()
      )}
    </ResourceList>
  );
}
