"use client";

import { ProjectNavigation } from "../navigations/project-navigation";
import { ProjectSubNavigation } from "../navigations/project-sub-navigation";
import { DeploymentsListControlCloud } from "./components/control-cloud";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsList } from "./components/table/deployments-list";

export default function Deployments({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <div>
      <ProjectNavigation projectId={projectId} />
      <ProjectSubNavigation onMount={() => {}} />
      <div className="flex flex-col">
        <DeploymentsListControls />
        <DeploymentsListControlCloud />
        <DeploymentsList />
      </div>
    </div>
  );
}
