"use client";

import { IconChevronLeftOutline18 } from "@unkey/icons";
import { Button, useStepWizard } from "@unkey/ui";
import { ProjectDataProvider } from "../../[appId]/(overview)/data-provider";
import { AddEnvVarsButton } from "../../_components/env-vars/add/add-env-vars-button";
import { EnvVarsSection } from "../../_components/env-vars/env-vars-section";
import { DeployAction } from "./deploy-action";

type EnvVarsStepProps = {
  projectId: string;
  appId: string;
  onDeploymentCreated: (deploymentId: string) => void;
};

export const EnvVarsStep = ({ projectId, appId, onDeploymentCreated }: EnvVarsStepProps) => {
  const { back } = useStepWizard();

  return (
    <>
      <Button
        variant="ghost"
        type="button"
        onClick={back}
        className="absolute top-3 left-3 z-50 flex items-center gap-1 hover:text-gray-11 group text-sm transition-colors text-gray-10"
      >
        <IconChevronLeftOutline18 className="!" />
        Back
      </Button>
      <div className="w-225">
        <ProjectDataProvider projectId={projectId} appId={appId}>
          <div className="flex flex-col gap-5">
            <div className="flex justify-end">
              <AddEnvVarsButton />
            </div>
            <EnvVarsSection />
          </div>
          <DeployAction
            projectId={projectId}
            appId={appId}
            onDeploymentCreated={onDeploymentCreated}
          />
        </ProjectDataProvider>
      </div>
    </>
  );
};
