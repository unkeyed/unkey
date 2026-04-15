"use client";
import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { DeploymentsListControlCloud } from "./components/control-cloud";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsCardList } from "./components/deployments-card-list";
import { DeploymentsHeader } from "./components/deployments-header";

export default function Deployments() {
  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <DeploymentsHeader />
      <DeploymentsListControls />
      <DeploymentsListControlCloud />
      <DeploymentsCardList />
    </ProjectContentWrapper>
  );
}
