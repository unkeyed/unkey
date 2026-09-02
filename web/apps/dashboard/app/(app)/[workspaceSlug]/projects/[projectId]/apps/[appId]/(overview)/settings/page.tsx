"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
} from "@unkey/ui";
import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";
import { useScrollToHash } from "./hooks/use-scroll-to-hash";

export default function SettingsPage() {
  const { bypass } = usePreventLeave();
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
