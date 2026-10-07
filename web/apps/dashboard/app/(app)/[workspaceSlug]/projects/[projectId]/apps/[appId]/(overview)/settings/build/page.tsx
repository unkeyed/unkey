"use client";

import { SettingsGroups } from "@unkey/ui";
import { BuildSettings } from "../deployment-settings";
import { EnvironmentSettingsProvider } from "../environment-provider";

export default function BuildSettingsPage() {
  return (
    <EnvironmentSettingsProvider>
      <SettingsGroups>
        <BuildSettings />
      </SettingsGroups>
    </EnvironmentSettingsProvider>
  );
}
