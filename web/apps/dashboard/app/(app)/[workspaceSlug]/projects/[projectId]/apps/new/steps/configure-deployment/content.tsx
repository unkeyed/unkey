"use client";

import { DeploymentSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/deployment-settings";
import { Button, useStepWizard } from "@unkey/ui";

export const ConfigureDeploymentContent = () => {
  const { next } = useStepWizard();

  return (
    <div className="w-225">
      <DeploymentSettings githubReadOnly sections={{ build: true }} />
      <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
        <Button type="button" variant="primary" size="xlg" className="rounded-lg" onClick={next}>
          Next
        </Button>
        <span className="text-gray-10 text-[13px] text-center">
          Start configuring your environment variables
        </span>
      </div>
    </div>
  );
};
