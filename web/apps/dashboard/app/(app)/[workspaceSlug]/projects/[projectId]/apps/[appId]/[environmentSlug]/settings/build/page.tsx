"use client";

import { SettingsGroups } from "@unkey/ui";
import { BuildSettings } from "../deployment-settings";

export default function BuildSettingsPage() {
  return (
    <SettingsGroups>
      <BuildSettings />
    </SettingsGroups>
  );
}
