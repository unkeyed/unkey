"use client";

import { SettingsDangerZone, SettingsGroups } from "@unkey/ui";
import { DeleteApp } from "../components/delete-app";
import { DisconnectGitHub } from "../components/disconnect-github";

export default function DangerSettingsPage() {
  return (
    <SettingsGroups>
      <SettingsDangerZone showTitle={false}>
        <DisconnectGitHub />
        <DeleteApp />
      </SettingsDangerZone>
    </SettingsGroups>
  );
}
