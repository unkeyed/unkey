"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import Link from "next/link";
import { useAppId, useProjectData } from "../../data-provider";
import { DeploymentsCardList } from "../../deployments/components/deployments-card-list";

const RECENT_LIMIT = 10;

export function RecentDeployments() {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const appId = useAppId();

  return (
    <DeploymentsCardList
      limit={RECENT_LIMIT}
      title="Deployments"
      headerAction={
        <Link
          href={routes.projects.apps.deployments({
            workspaceSlug: workspace.slug,
            projectId,
            appId,
          })}
          className="text-[13px] text-gray-11 hover:text-gray-12 transition-colors"
        >
          View all deployments
        </Link>
      }
    />
  );
}
