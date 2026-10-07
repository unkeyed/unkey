"use client";

import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { SettingsDangerZone, SettingsGroups } from "@unkey/ui";
import { LoadError } from "../../components/load-error";
import { DeleteApp } from "./components/delete-app";
import { DisconnectGitHub } from "./components/disconnect-github";
import { SettingsSkeleton } from "./components/settings-skeleton";
import { GeneralSettings } from "./deployment-settings";
import { useApp } from "./hooks/use-build-source";

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
      <GeneralSettings />
      <SettingsDangerZone>
        <DisconnectGitHub />
        <DeleteApp />
      </SettingsDangerZone>
    </SettingsGroups>
  );
}
