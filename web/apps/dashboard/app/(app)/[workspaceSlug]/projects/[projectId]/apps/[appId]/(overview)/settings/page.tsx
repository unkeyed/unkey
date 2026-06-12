"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  SettingsDangerZone,
} from "@unkey/ui";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";

export default function SettingsPage() {
  const { bypass } = usePreventLeave();

  return (
    <EnvironmentSettingsProvider>
      <PageContainer>
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Settings</PageHeaderTitle>
            <PageHeaderDescription>
              Review the defaults. Edit anything you'd like to adjust.
            </PageHeaderDescription>
          </PageHeaderContent>
        </PageHeader>
        <PageBody className="flex flex-col gap-6 pt-6 pb-14">
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
