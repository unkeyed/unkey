"use client";

import { EnvVarsGroup } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/env-vars/env-vars-group";
import { ADVANCED_SETTINGS } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/advanced-settings/advanced-settings";
import { BuildSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/build-settings";
import { ComputeSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/compute/compute-settings";
import { RUNTIME_SETTINGS } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/runtime-settings/runtime-settings";
import { SettingsPageCollapsible } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/settings-page";
import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import {
  SettingsGroups,
  UnsavedChangesScope,
  useReportUnsavedChanges,
  useUnsavedChanges,
} from "@unkey/ui";
import { DeployAction } from "./deploy-action";
import { ConfigureFrame } from "./frame";

export function ConfigureDeploymentContent({
  onDeploymentCreated,
}: { onDeploymentCreated: (deploymentId: string) => void }) {
  const { isDirty, report } = useUnsavedChanges();
  useReportUnsavedChanges(isDirty);

  return (
    <ConfigureFrame>
      <UnsavedChangesScope report={report}>
        <SettingsGroups pendingNote={NEXT_DEPLOY} className="gap-4 p-0">
          <BuildSettings githubReadOnly />
          <ComputeSettings />
          <EnvVarsGroup />
          <SettingsPageCollapsible page={RUNTIME_SETTINGS} />
          <SettingsPageCollapsible page={ADVANCED_SETTINGS} />
        </SettingsGroups>
      </UnsavedChangesScope>
      <DeployAction dirty={isDirty} onDeploymentCreated={onDeploymentCreated} />
    </ConfigureFrame>
  );
}
