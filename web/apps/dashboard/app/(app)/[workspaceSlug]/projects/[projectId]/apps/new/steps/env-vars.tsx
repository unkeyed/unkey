"use client";

import { IconChevronLeftOutline18 } from "@unkey/icons";
import { Button, useStepWizard } from "@unkey/ui";
import { useState } from "react";
import {
  ProjectDataProvider,
  useAppId,
  useProjectData,
} from "../../[appId]/(overview)/data-provider";
import { AddEnvVarsButton } from "../../[appId]/(overview)/env-vars/components/add/add-env-vars-button";
import { EnvVarsList } from "../../[appId]/(overview)/env-vars/components/list/env-vars-list";
import {
  EnvVarsToolbar,
  type SortOption,
} from "../../[appId]/(overview)/env-vars/components/toolbar/env-vars-toolbar";
import { ALL_ENVIRONMENTS } from "../../_components/env-vars/shared/environment-select";
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

function DeploymentEnvVars() {
  const { projectId, environments } = useProjectData();
  const appId = useAppId();
  const [environmentFilter, setEnvironmentFilter] = useState(ALL_ENVIRONMENTS);
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");

  return (
    <div className="flex flex-col gap-5">
      <div className="flex justify-end">
        <AddEnvVarsButton />
      </div>
      <EnvVarsToolbar
        environmentFilter={environmentFilter}
        onEnvironmentFilterChange={setEnvironmentFilter}
        sortBy={sortBy}
        onSortChange={setSortBy}
      />
      <EnvVarsList
        projectId={projectId}
        appId={appId}
        environments={environments}
        environmentFilter={environmentFilter}
        sortBy={sortBy}
      />
    </div>
  );
}
