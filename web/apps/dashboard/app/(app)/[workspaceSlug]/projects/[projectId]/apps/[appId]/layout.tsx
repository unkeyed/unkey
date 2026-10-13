"use client";
import { LeaveGuard } from "@/components/leave-guard";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { useParams } from "next/navigation";
import type { PropsWithChildren } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useParams<{ projectId: string }>();

  return (
    <ProjectDataProvider>
      <LeaveGuard
        fallbackHref={routes.projects.detail({ workspaceSlug: workspace.slug, projectId })}
      >
        <div className="h-full flex flex-col overflow-hidden">
          <div className="flex flex-1 min-h-0">
            <div className="flex-1 overflow-auto">{children}</div>
          </div>
          <PendingRedeployBanner />
        </div>
      </LeaveGuard>
    </ProjectDataProvider>
  );
}
