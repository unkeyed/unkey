"use client";

import { DeploymentSettings } from "./deployment-settings";
import { EnvironmentSettingsProvider } from "./environment-provider";

export default function SettingsPage() {
  return (
    <EnvironmentSettingsProvider>
      <div className="w-225 flex flex-col justify-center items-center gap-6 mx-auto my-14">
        <div className="flex flex-col gap-2 items-center">
          <span className="font-semibold text-gray-12 leading-8 text-lg">Configure deployment</span>
          <span className="leading-4 text-gray-11 text-[13px]">
            Review the defaults. Edit anything you'd like to adjust.
          </span>
        </div>
        <DeploymentSettings />
      </div>
    </EnvironmentSettingsProvider>
  );
}
