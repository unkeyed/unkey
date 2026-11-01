"use client";

import { Switch } from "@/components/ui/switch";
import { SettingsRow, SettingsRowContent, SettingsRowHeader, SettingsRowTitle } from "@unkey/ui";
import { useEnvironmentSettings } from "../../../../../_components/settings/environment-provider";

export function AutoDeploy() {
  const { settings, update } = useEnvironmentSettings();
  const toggle = (autoDeploy: boolean) => {
    // The failure is toasted and the switch rolls back
    update(
      (draft) => {
        draft.autoDeploy = autoDeploy;
      },
      { optimistic: true },
    ).catch(() => undefined);
  };

  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Auto-deploy</SettingsRowTitle>
      </SettingsRowHeader>
      <SettingsRowContent>
        <Switch checked={settings.autoDeploy} onCheckedChange={toggle} aria-label="Auto-deploy" />
      </SettingsRowContent>
    </SettingsRow>
  );
}
