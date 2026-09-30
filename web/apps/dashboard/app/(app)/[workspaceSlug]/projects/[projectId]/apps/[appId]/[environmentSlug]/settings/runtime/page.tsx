"use client";

import { SettingsGroups } from "@unkey/ui";
import { RuntimeSettings } from "../deployment-settings";

export default function RuntimeSettingsPage() {
  return (
    <SettingsGroups>
      <RuntimeSettings />
    </SettingsGroups>
  );
}
