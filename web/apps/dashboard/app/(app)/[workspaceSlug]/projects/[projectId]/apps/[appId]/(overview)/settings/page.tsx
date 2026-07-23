"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
} from "@unkey/ui";
import { useAppId, useProjectData } from "../data-provider";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";
import { useScrollToHash } from "./hooks/use-scroll-to-hash";

export default function SettingsPage() {
  const { bypass } = usePreventLeave();
  const appId = useAppId();
  const { projectId } = useProjectData();
  const { data: apps } = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = apps.at(0);
  useScrollToHash();

  if (!app) {
    return null;
  }

  return (
    <EnvironmentSettingsProvider>
      <PageContainer>
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>App Settings</PageHeaderTitle>
          </PageHeaderContent>
        </PageHeader>
        <PageBody>
          <DeploymentSettings onBeforeNavigate={bypass} sourceType={app.sourceType} app={app} />
          <SettingsDangerZone>
            {app.sourceType === "github" && <DisconnectGitHub />}
            <DeleteApp />
          </SettingsDangerZone>
        </PageBody>
      </PageContainer>
    </EnvironmentSettingsProvider>
  );
}
