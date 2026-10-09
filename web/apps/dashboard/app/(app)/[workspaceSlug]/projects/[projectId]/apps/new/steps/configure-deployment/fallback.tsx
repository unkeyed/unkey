"use client";

import { SettingsSkeleton } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/settings-skeleton";
import { DeployAction } from "./deploy-action";
import { ConfigureFrame } from "./frame";

export function ConfigureDeploymentFallback({
  onDeploymentCreated,
}: { onDeploymentCreated: (deploymentId: string) => void }) {
  return (
    <ConfigureFrame>
      <SettingsSkeleton className="p-0" />
      <DeployAction state="loading" onDeploymentCreated={onDeploymentCreated} />
    </ConfigureFrame>
  );
}
