"use client";

import { ProjectLayout } from "../project-layout";
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
      <ProjectLayout projectId={projectId}>
        <div className="flex flex-col">
          <DeploymentsListControls />
          <DeploymentsListControlCloud />
          <DeploymentsList />
        </div>
      </ProjectLayout>
    </div>
  );
}
