"use client";

import { ADVANCED_SETTINGS } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/advanced-settings/advanced-settings";
import { BuildSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/build-settings";
import { ComputeSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/compute/compute-settings";
import { RUNTIME_SETTINGS } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/runtime-settings/runtime-settings";
import { SettingsPageCollapsible } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/settings-page";
import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { Button, SettingsGroups, useStepWizard } from "@unkey/ui";

export function ConfigureDeploymentContent() {
  const { next } = useStepWizard();

  return (
    <div className="w-225">
      <SettingsGroups pendingNote={NEXT_DEPLOY} className="gap-4 p-0">
        <BuildSettings githubReadOnly />
        <ComputeSettings />
        <SettingsPageCollapsible page={RUNTIME_SETTINGS} />
        <SettingsPageCollapsible page={ADVANCED_SETTINGS} />
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
}
