"use client";

import { DeploymentsNavigation } from "../navigation";
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
      <DeploymentsNavigation projectId={projectId} />
      <div className="flex flex-col">
        <DeploymentsListControls />
        <DeploymentsListControlCloud />
        <DeploymentsList />
      </div>
    </div>
  );
}
