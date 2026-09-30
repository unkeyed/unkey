"use client";

import {
  AdvancedSettings,
  BuildSettings,
  ComputeSettings,
  RuntimeSettings,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/[environmentSlug]/settings/deployment-settings";
import { Button, SettingsGroups, useStepWizard } from "@unkey/ui";

export const ConfigureDeploymentContent = () => {
  const { next } = useStepWizard();

  return (
    <div className="w-225">
      <SettingsGroups className="p-0">
        <BuildSettings githubReadOnly />
        <RuntimeSettings />
        <AdvancedSettings />
        <ComputeSettings />
      </SettingsGroups>
      <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
        <Button type="button" variant="primary" size="xlg" className="rounded-lg" onClick={next}>
          Next
        </Button>
        <span className="text-gray-10 text-sm text-center">
          Start configuring your environment variables
        </span>
      </div>
    </div>
  );
};
