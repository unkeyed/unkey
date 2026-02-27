"use client";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { DeploymentProgressSection } from "./(overview)/components/sections/deployment-progress-section";

export default function DeploymentOverview() {
  return (
    <ProjectContentWrapper centered>
      <DeploymentProgressSection />
    </ProjectContentWrapper>
  );
}
