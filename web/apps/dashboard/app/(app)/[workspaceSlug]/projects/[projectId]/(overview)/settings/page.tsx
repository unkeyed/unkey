"use client";

import { SettingsShell } from "@unkey/ui";
import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { DeleteProject } from "./components/delete-project";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";
import { PendingRedeployBanner } from "./pending-redeploy-banner";

export default function SettingsPage() {
  const { bypass } = usePreventLeave();

  return (
    <EnvironmentSettingsProvider>
      <SettingsShell>
        <div className="flex flex-col gap-2 items-center">
          <span className="font-semibold text-gray-12 leading-8 text-lg">Configure deployment</span>
          <span className="leading-4 text-gray-11 text-[13px]">
            Review the defaults. Edit anything you'd like to adjust.
          </span>
        </div>
        <DeploymentSettings onBeforeNavigate={bypass} />
      </SettingsShell>
      <PendingRedeployBanner />
      <DeleteProject />
    </EnvironmentSettingsProvider>
  );
}
