"use client";

import { useParams } from "next/navigation";
import { DeploymentsListControlCloud } from "./components/control-cloud";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsList } from "./components/table/deployments-list";

export default function Deployments() {

  // biome-ignore lint/style/noNonNullAssertion: shut up nextjs
  const { projectId } = useParams<{ projectId: string }>()!;
  return (
    <div className="flex flex-col">
      <DeploymentsListControls />
      <DeploymentsListControlCloud />
      <DeploymentsList projectId={projectId} />
    </div>
  );
}
