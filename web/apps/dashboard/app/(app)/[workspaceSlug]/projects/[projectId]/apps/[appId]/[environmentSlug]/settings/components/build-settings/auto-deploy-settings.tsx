"use client";

import { collection } from "@/lib/collections";
import { getErrorToast } from "@/lib/unkey-client";
import { toast } from "@unkey/ui";
import { useState } from "react";
import { useAppEnvironment } from "../../../environment-context";
import { useEnvironmentSettings } from "../../environment-provider";
import { SettingsToggleRow } from "../shared/settings-toggle-row";

export function AutoDeploy() {
  const { settings } = useEnvironmentSettings();
  const { environment } = useAppEnvironment();
  const [pending, setPending] = useState(false);

  const environmentName = environment.slug.charAt(0).toUpperCase() + environment.slug.slice(1);

  const onCheckedChange = async (autoDeploy: boolean) => {
    setPending(true);
    const tx = collection.environmentSettings.update(
      settings.environmentId,
      { metadata: { silent: true } },
      (draft) => {
        draft.autoDeploy = autoDeploy;
      },
    );
    try {
      await tx.isPersisted.promise;
      toast.success(`Auto-deploy turned ${autoDeploy ? "on" : "off"} for ${environmentName}`);
    } catch (error) {
      const { message, description } = getErrorToast(error, "Failed to update auto-deploy");
      toast.error(message, { description });
    } finally {
      setPending(false);
    }
  };

  return (
    <SettingsToggleRow
      title="Auto-deploy"
      description="Deploy automatically when you push to GitHub."
      checked={settings.autoDeploy}
      onCheckedChange={onCheckedChange}
      pending={pending}
    />
  );
}
