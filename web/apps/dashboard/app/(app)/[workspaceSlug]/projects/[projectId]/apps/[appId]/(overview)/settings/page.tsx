"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
} from "@unkey/ui";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";
import { useScrollToHash } from "./hooks/use-scroll-to-hash";

export default function SettingsPage() {
  const { bypass } = usePreventLeave();
  useScrollToHash();

  return (
    <EnvironmentSettingsProvider>
      <PageContainer>
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>App Settings</PageHeaderTitle>
          </PageHeaderContent>
        </PageHeader>
        <PageBody>
          <DeploymentSettings onBeforeNavigate={bypass} />
          <SettingsDangerZone>
            <DisconnectGitHub />
            <DeleteApp />
          </SettingsDangerZone>
        </PageBody>
      </PageContainer>
    </EnvironmentSettingsProvider>
  );
}
