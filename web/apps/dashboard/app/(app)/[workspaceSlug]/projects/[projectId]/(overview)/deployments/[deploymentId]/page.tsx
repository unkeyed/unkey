"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProject } from "../../layout-provider";
import { DeploymentDomainsSection } from "./(overview)/components/sections/deployment-domains-section";
import { DeploymentInfoSection } from "./(overview)/components/sections/deployment-info-section";
import { DeploymentLogsSection } from "./(overview)/components/sections/deployment-logs-section";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";

export default function DeploymentOverview() {


  return (
    <ProjectContentWrapper centered>
      <DeploymentInfoSection
      />
      <DeploymentDomainsSection />
      <DeploymentNetworkSection />
      <DeploymentLogsSection />
    </ProjectContentWrapper>
  );
}
