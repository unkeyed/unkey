"use client";

import { SettingsGroups } from "@unkey/ui";
import { ComputeSettings } from "./deployment-settings";

export default function ComputeSettingsPage() {
  return (
    <SettingsGroups>
      <ComputeSettings />
    </SettingsGroups>
  );
}
