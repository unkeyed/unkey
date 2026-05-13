"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { SettingsDangerZone, SettingsShell } from "@unkey/ui";
import { DeleteProject } from "./components/delete-project";
import { DisconnectGitHub } from "./components/disconnect-github";
import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";
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
      <div className="w-225 mx-auto mt-10 mb-14">
        <SettingsDangerZone>
          <DisconnectGitHub />
          <DeleteProject />
        </SettingsDangerZone>
      </div>
    </EnvironmentSettingsProvider>
  );
}
