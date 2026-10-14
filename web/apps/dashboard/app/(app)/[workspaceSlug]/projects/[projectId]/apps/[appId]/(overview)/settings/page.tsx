"use client";

import { LoadError } from "@/components/load-error";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { SettingsDangerZone, SettingsGroup, SettingsGroupContent, SettingsGroups } from "@unkey/ui";
import { useApp } from "../../../_components/settings/hooks/use-app";
import { SettingsSkeleton } from "../../../_components/settings/settings-skeleton";
import { AppName } from "./components/app-name";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";

export default function GeneralSettingsPage() {
  const { app, isLoading } = useApp();
  const appsLoad = useCollectionLoad(collection.apps.utils);
  if (!app) {
    if (appsLoad.failed && !isLoading) {
      return <LoadError title="Could not load this app" onRetry={appsLoad.retry} />;
    }
    return <SettingsSkeleton />;
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
