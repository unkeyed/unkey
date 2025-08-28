"use client";

import { DeploymentsListControlCloud } from "./components/control-cloud";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsList } from "./components/table/deployments-list";

export default function Deployments() {
  return (
    <div className="flex flex-col">
      <DeploymentsListControls />
      <DeploymentsListControlCloud />
      <DeploymentsList />
    </div>
  );
}
