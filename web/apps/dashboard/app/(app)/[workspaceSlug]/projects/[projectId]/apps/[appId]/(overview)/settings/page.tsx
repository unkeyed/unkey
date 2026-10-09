"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import {
  DiscardChangesDialog,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
} from "@unkey/ui";
import { useProjectData } from "../data-provider";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";
import { useScrollToHash } from "./hooks/use-scroll-to-hash";

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const { bypass, leavePrompt } = usePreventLeave(
    true,
    routes.projects.detail({ workspaceSlug: workspace.slug, projectId }),
  );
  useScrollToHash();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>App Settings</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <EnvironmentSettingsProvider>
          <DeploymentSettings onBeforeNavigate={bypass} />
        </EnvironmentSettingsProvider>
        <SettingsDangerZone>
          <DisconnectGitHub />
          <DeleteApp />
        </SettingsDangerZone>
      </PageBody>
      <DiscardChangesDialog {...leavePrompt} />
    </PageContainer>
  );
}
