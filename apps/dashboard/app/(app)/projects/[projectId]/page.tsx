"use client";

import { DeploymentsListControlCloud } from "./deployments/components/control-cloud";
import { DeploymentsListControls } from "./deployments/components/controls";
import { DeploymentsList } from "./deployments/components/table/deployments-list";
import { DeploymentsNavigation } from "./deployments/navigation";

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
