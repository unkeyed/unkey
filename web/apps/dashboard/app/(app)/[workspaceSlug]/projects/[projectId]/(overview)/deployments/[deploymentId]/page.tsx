"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { Cube } from "@unkey/icons";
import { Card } from "@unkey/ui";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { ScrollToBottomButton } from "./(overview)/components/scroll-to-bottom-button";
import { DeploymentDomainsSection } from "./(overview)/components/sections/deployment-domains-section";
import { DeploymentInfoSection } from "./(overview)/components/sections/deployment-info-section";
import { DeploymentLogsSection } from "./(overview)/components/sections/deployment-logs-section";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { DeploymentBuildStepsTable } from "./(overview)/components/table/deployment-build-steps-table";

export default function DeploymentOverview() {


  return (
    <>
      <ProjectContentWrapper centered>
        <DeploymentInfoSection />
        <DeploymentDomainsSection />
        <DeploymentNetworkSection />
        <DeploymentLogsSection />
      </ProjectContentWrapper>
      <ScrollToBottomButton />
    </>
  );
}
