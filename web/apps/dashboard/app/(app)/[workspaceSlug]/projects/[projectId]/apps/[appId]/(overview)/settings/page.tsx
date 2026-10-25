"use client";

import { ResourceNotifications } from "@/app/(app)/[workspaceSlug]/settings/notifications/resource-notifications";
import { LoadError } from "@/components/load-error";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import {
  SettingsDangerZone,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroups,
  SettingsPageSkeleton,
} from "@unkey/ui";
import { useApp } from "../../../_components/settings/hooks/use-app";
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
    return <GeneralSettingsSkeleton />;
  }

  return (
    <SettingsGroups>
      <SettingsGroup>
        <SettingsGroupContent>
          <AppName appId={app.id} name={app.name} />
        </SettingsGroupContent>
      </SettingsGroup>
      <ResourceNotifications projectId={app.projectId} appId={app.id} />
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
