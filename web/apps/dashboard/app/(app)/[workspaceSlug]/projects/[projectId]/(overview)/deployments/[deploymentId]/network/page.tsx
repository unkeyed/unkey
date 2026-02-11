"use client";
import { useProjectData } from "../../../data-provider";
import { DeploymentNetworkView } from "./deployment-network-view";

export default function DeploymentDetailsPage() {
  const { projectId, project } = useProjectData();
  const liveDeploymentId = project?.liveDeploymentId;

  return <DeploymentNetworkView projectId={projectId} deploymentId={liveDeploymentId} />;
}
