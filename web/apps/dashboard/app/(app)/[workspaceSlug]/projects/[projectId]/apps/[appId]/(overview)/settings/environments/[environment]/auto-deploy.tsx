"use client";

import { Switch } from "@/components/ui/switch";
import { SettingsRow, SettingsRowContent, SettingsRowHeader, SettingsRowTitle } from "@unkey/ui";
import { useEnvironmentSettings } from "../../../../../_components/settings/environment-provider";

export function AutoDeploy() {
  const { settings, update } = useEnvironmentSettings();
  const toggle = (autoDeploy: boolean) => {
    const tx = update(
      (draft) => {
        draft.autoDeploy = autoDeploy;
      },
      { optimistic: true },
    );
    // The collection toasts the failure and rolls the switch back.
    tx.isPersisted.promise.catch(() => undefined);
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
