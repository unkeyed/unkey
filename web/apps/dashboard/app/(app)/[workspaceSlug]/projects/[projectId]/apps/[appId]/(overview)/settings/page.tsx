"use client";

import { LoadError } from "@/components/load-error";
import { useApp } from "@/lib/deploy-data";
import {
  SettingsDangerZone,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroups,
  SettingsPageSkeleton,
} from "@unkey/ui";
import { AppName } from "./components/app-name";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";

export default function GeneralSettingsPage() {
  const { data: app, isLoading, isError, refetch } = useApp();
  if (!app) {
    if (isError && !isLoading) {
      return <LoadError title="Could not load this app" onRetry={refetch} />;
    }
    return <GeneralSettingsSkeleton />;
  }

  return (
    <SettingsGroups>
      <SettingsGroup>
        <SettingsGroupContent>
          <AppName appId={app.id} name={app.name} />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsDangerZone>
        <DisconnectGitHub />
        <DeleteApp />
      </SettingsDangerZone>
    </SettingsGroups>
  );
}

function GeneralSettingsSkeleton() {
  return (
    <SettingsGroups>
      <SettingsPageSkeleton groups={[{ rows: [{ descriptionLines: 0 }] }]} dangerZone={["w-30"]} />
    </SettingsGroups>
  );
}
