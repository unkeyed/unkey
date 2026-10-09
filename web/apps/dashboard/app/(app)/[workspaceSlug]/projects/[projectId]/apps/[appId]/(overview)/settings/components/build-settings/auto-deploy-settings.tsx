"use client";

import { Switch } from "@/components/ui/switch";
import { getErrorToast } from "@/lib/unkey-client";
import {
  Loading,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { useEnvironmentSettings } from "../../../../../_components/settings/environment-provider";

export function AutoDeploy({ environmentName }: { environmentName: string }) {
  const { settings, update } = useEnvironmentSettings();
  const [pending, setPending] = useState(false);

  const onCheckedChange = async (autoDeploy: boolean) => {
    setPending(true);
    const tx = update((draft) => {
      draft.autoDeploy = autoDeploy;
    });
    try {
      await tx.isPersisted.promise;
      toast.success(`Auto-deploy turned ${autoDeploy ? "on" : "off"} for ${environmentName}`);
    } catch (error) {
      const toastContent = getErrorToast(error, "Failed to update auto-deploy");
      toast.error(toastContent.message, { description: toastContent.description });
    } finally {
      setPending(false);
    }
  };

  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Auto-deploy</SettingsRowTitle>
      </SettingsRowHeader>
      <SettingsRowContent>
        <div className="flex items-center gap-2">
          <Switch
            checked={settings.autoDeploy}
            onCheckedChange={onCheckedChange}
            disabled={pending}
            aria-label="Auto-deploy"
          />
          {pending ? <Loading size={14} className="text-gray-11" /> : null}
        </div>
      </SettingsRowContent>
    </SettingsRow>
  );
}
