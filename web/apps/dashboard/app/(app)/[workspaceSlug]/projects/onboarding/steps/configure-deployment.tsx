"use client";

import { Button } from "@unkey/ui";
import { ProjectDataProvider } from "../../[projectId]/(overview)/data-provider";
import { DeploymentSettings } from "../../[projectId]/(overview)/settings/deployment-settings";
import { EnvironmentSettingsProvider } from "../../[projectId]/(overview)/settings/environment-provider";

type ConfigureDeploymentStepProps = {
  projectId: string;
};

export const ConfigureDeploymentStep = ({ projectId }: ConfigureDeploymentStepProps) => {
  return (
    <ProjectDataProvider projectId={projectId}>
      <EnvironmentSettingsProvider>
        <div className="w-[900px]">
          <DeploymentSettings githubReadOnly />
          <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
            <Button type="submit" variant="primary" size="xlg" className="rounded-lg">
              Deploy
            </Button>
            <span className="text-gray-10 text-[13px] text-center">
              We’ll build your image, provision infrastructure, and more.
              <br />
            </span>
          </div>
        </div>
      </EnvironmentSettingsProvider>
    </ProjectDataProvider>
  );
};
