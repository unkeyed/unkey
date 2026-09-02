"use client";

import { Button, useStepWizard } from "@unkey/ui";
import { IconChevronLeftOutline18 } from "nucleo-ui-outline-18";
import { ProjectDataProvider } from "../../[appId]/(overview)/data-provider";
import { DeploymentEnvVars } from "../../[appId]/(overview)/env-vars/deployment-env-vars";
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
        className="absolute top-3 left-3 z-50 flex items-center gap-1 hover:text-gray-11 group text-[13px] transition-colors text-gray-10"
      >
        <IconChevronLeftOutline18 className="!" />
        Back
      </Button>
      <div className="w-225">
        <ProjectDataProvider projectId={projectId} appId={appId}>
          <DeploymentEnvVars />
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
