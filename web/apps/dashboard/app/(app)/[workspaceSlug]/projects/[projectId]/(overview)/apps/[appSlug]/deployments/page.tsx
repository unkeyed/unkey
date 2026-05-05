"use client";

import { ProjectContentWrapper } from "../../../../components/project-content-wrapper";
import { DeploymentsListControls } from "../../../deployments/components/controls";
import { DeploymentsCardList } from "../../../deployments/components/deployments-card-list";
import { DeploymentsHeader } from "../../../deployments/components/deployments-header";

/**
 * App → Deployments list. The middle "Manage" sidebar lives in the
 * sibling layout so it persists into the deployment detail route.
 */
export default function AppDeploymentsPage() {
  return (
    <ProjectContentWrapper centered className="pt-12">
      <DeploymentsHeader />
      <DeploymentsListControls />
      <DeploymentsCardList />
    </ProjectContentWrapper>
  );
}
