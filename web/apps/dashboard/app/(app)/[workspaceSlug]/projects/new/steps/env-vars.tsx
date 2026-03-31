"use client";

import { ChevronLeft } from "@unkey/icons";
import { Button, useStepWizard } from "@unkey/ui";
import { ProjectDataProvider } from "../../[projectId]/(overview)/data-provider";
import { DeploymentEnvVars } from "../../[projectId]/(overview)/env-vars/deployment-env-vars";
import { DeployAction } from "./deploy-action";

type EnvVarsStepProps = {
  projectId: string;
  onDeploymentCreated: (deploymentId: string) => void;
};

export const EnvVarsStep = ({ projectId, onDeploymentCreated }: EnvVarsStepProps) => {
  const { back } = useStepWizard();

  return (
    <>
      <Button
        variant="ghost"
        type="button"
        onClick={back}
        className="absolute top-3 left-3 z-50 flex items-center gap-1 hover:text-gray-11 group text-[13px] transition-colors text-gray-10"
      >
        <ChevronLeft className="size-3!" iconSize="sm-regular" />
        Back
      </Button>
      <div className="w-225">
        <ProjectDataProvider projectId={projectId}>
          <DeploymentEnvVars />
          <DeployAction projectId={projectId} onDeploymentCreated={onDeploymentCreated} />
        </ProjectDataProvider>
      </div>
    </>
  );
};
