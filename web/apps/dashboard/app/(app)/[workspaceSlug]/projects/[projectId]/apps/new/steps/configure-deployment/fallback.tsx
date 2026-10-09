"use client";

import { SettingsSkeleton } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/settings-skeleton";

export function ConfigureDeploymentFallback() {
  return (
    <div className="w-225">
      <SettingsSkeleton className="p-0" />
    </div>
  );
}
