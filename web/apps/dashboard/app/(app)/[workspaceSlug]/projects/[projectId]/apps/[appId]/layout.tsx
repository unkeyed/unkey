"use client";
import { GuardedNavigateContext, usePreventLeave } from "@/hooks/use-prevent-leave";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { DiscardChangesDialog, UnsavedChangesScope, useUnsavedChanges } from "@unkey/ui";
import { useParams } from "next/navigation";
import type { PropsWithChildren } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useParams<{ projectId: string }>();
  const { isDirty, report } = useUnsavedChanges();
  const { leavePrompt, navigate } = usePreventLeave(
    isDirty,
    routes.projects.detail({ workspaceSlug: workspace.slug, projectId }),
  );

  return (
    <ProjectDataProvider>
      <UnsavedChangesScope report={report}>
        <GuardedNavigateContext value={navigate}>
          <div className="flex flex-1 flex-col">
            <div className="flex flex-1">
              <div className="min-w-0 flex-1">{children}</div>
            </div>
            <PendingRedeployBanner />
          </div>
        </GuardedNavigateContext>
      </UnsavedChangesScope>
      <DiscardChangesDialog {...leavePrompt} />
    </ProjectDataProvider>
  );
}
