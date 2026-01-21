"use client";
import { useProject } from "../../../layout-provider";
import { DeploymentNetworkView } from "./deployment-network-view";

export default function DeploymentDetailsPage() {
  const { projectId, liveDeploymentId } = useProject();

  return <DeploymentNetworkView projectId={projectId} liveDeploymentId={liveDeploymentId} />;
}
