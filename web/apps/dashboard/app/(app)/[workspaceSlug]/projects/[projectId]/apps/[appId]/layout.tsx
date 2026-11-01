"use client";
import { LeaveGuard } from "@/components/leave-guard";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { DeployDataProvider } from "@/lib/deploy-data";
import { routes } from "@/lib/navigation/routes";
import { useParams } from "next/navigation";
import type { PropsWithChildren } from "react";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useParams<{ projectId: string }>();

  return (
    <DeployDataProvider>
      <LeaveGuard
        fallbackHref={routes.projects.detail({ workspaceSlug: workspace.slug, projectId })}
      >
        <div className="flex flex-1 flex-col">
          <div className="flex flex-1">
            <div className="min-w-0 flex-1">{children}</div>
          </div>
          <PendingRedeployBanner />
        </div>
      </LeaveGuard>
    </DeployDataProvider>
  );
}
