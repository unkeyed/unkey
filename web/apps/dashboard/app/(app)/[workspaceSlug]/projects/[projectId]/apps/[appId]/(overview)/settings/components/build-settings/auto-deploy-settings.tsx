"use client";

import { Switch } from "@/components/ui/switch";
import { collection } from "@/lib/collections";
import { getErrorToast } from "@/lib/unkey-client";
import {
  Loading,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
  toast,
} from "@unkey/ui";
import { useId, useState } from "react";
import { useEnvironmentSettings } from "../../environment-provider";

export function AutoDeploy({ environmentName }: { environmentName: string }) {
  const { settings } = useEnvironmentSettings();
  const [pending, setPending] = useState(false);
  const titleId = useId();

  const onCheckedChange = async (autoDeploy: boolean) => {
    setPending(true);
    const tx = collection.environmentSettings.update(settings.environmentId, (draft) => {
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
    <SettingsRow className="flex-row items-center justify-between gap-6">
      <SettingsRowHeader className="min-w-0 shrink">
        <SettingsRowTitle id={titleId}>Auto-deploy</SettingsRowTitle>
      </SettingsRowHeader>
      <SettingsRowContent className="flex flex-none items-center justify-end gap-2">
        <Switch
          checked={settings.autoDeploy}
          onCheckedChange={onCheckedChange}
          disabled={pending}
          aria-labelledby={titleId}
        />
        {pending ? <Loading size={14} className="text-gray-11" /> : null}
      </SettingsRowContent>
    </SettingsRow>
  );
}
