"use client";

import { SettingsGroups } from "@unkey/ui";
import { AdvancedSettings } from "../deployment-settings";
import { EnvironmentSettingsProvider } from "../environment-provider";

export default function AdvancedSettingsPage() {
  return (
    <EnvironmentSettingsProvider>
      <SettingsGroups>
        <AdvancedSettings />
      </SettingsGroups>
    </EnvironmentSettingsProvider>
  );
}
