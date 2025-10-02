"use client";
import { ProjectContentWrapper } from "../components/project-content-wrapper";
import { DeploymentsListControlCloud } from "./components/control-cloud";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsList } from "./components/table/deployments-list";

export default function Deployments() {
  return (
    <ProjectContentWrapper>
      <DeploymentsListControls />
      <DeploymentsListControlCloud />
      <DeploymentsList />
    </ProjectContentWrapper>
  );
}
