"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { ResourceList, ResourceListHeader } from "@unkey/ui";
import Link from "next/link";
import { useAppId, useProjectData } from "../../data-provider";
import { DeploymentsCardList } from "../../deployments/components/deployments-card-list";

const RECENT_LIMIT = 10;

export function RecentDeployments() {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const appId = useAppId();

  return (
    <ResourceList>
      <ResourceListHeader className="flex-row items-center justify-between">
        <h2 className="font-medium text-accent-12 text-sm">Deployments</h2>
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
      <DeploymentsCardList limit={RECENT_LIMIT} />
    </ResourceList>
  );
}
