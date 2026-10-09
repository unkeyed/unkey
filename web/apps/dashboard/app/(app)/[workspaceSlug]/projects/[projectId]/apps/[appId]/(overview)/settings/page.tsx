"use client";

import { useLeaveBypass } from "@/components/leave-guard";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
  useReportUnsavedChanges,
} from "@unkey/ui";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";
import { useScrollToHash } from "./hooks/use-scroll-to-hash";

export default function SettingsPage() {
  // The settings forms here do not report their own dirty state.
  useReportUnsavedChanges(true);
  const bypass = useLeaveBypass();
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
    </PageContainer>
  );
}
