"use client";

import { DeploymentsListControlCloud } from "./components/control-cloud";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsList } from "./components/table/deployments-list";
import { DeploymentsNavigation } from "./navigation";

export default function Deployments({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <div>
      <DeploymentsNavigation projectId={projectId} />
      <div className="flex flex-col">
        <DeploymentsListControls />
        <DeploymentsListControlCloud />
        <DeploymentsList />
      </div>
    </div>
  );
}
