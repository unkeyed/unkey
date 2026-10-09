"use client";

import { SettingsGroup, SettingsGroupContent, SettingsGroupTitle } from "@unkey/ui";
import { Command } from "../../../../../_components/settings/runtime-settings/command";
import { Healthcheck } from "../../../../../_components/settings/runtime-settings/healthcheck";
import { Port } from "../../../../../_components/settings/runtime-settings/port-settings";

export function RuntimeSettings() {
  return (
    <>
      <SettingsGroup>
        <SettingsGroupTitle>Process</SettingsGroupTitle>
        <SettingsGroupContent>
          <Port />
          <Command />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsGroup>
        <SettingsGroupTitle>Health check</SettingsGroupTitle>
        <SettingsGroupContent>
          <Healthcheck />
        </SettingsGroupContent>
      </SettingsGroup>
    </>
  );
}
