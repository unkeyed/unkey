"use client";

import { SettingsGroups } from "@unkey/ui";
import { DeploySettings } from "../deployment-settings";

export default function DeploysSettingsPage() {
  return (
    <SettingsGroups>
      <DeploySettings />
    </SettingsGroups>
  );
}
