"use client";

import { SettingsGroups } from "@unkey/ui";
import { RuntimeSettings } from "../deployment-settings";
import { EnvironmentSettingsProvider } from "../environment-provider";

export default function RuntimeSettingsPage() {
  return (
    <EnvironmentSettingsProvider>
      <SettingsGroups>
        <RuntimeSettings />
      </SettingsGroups>
    </EnvironmentSettingsProvider>
  );
}
