"use client";
import { Cloud, Earth } from "@unkey/icons";
import { EmptySection } from "./(overview)/components/empty-section";
import { useProjectData } from "./(overview)/data-provider";
import { DeploymentInfo } from "./(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-info";
import { DeploymentLayoutProvider } from "./(overview)/deployments/[deploymentId]/layout-provider";
import { ActiveDeploymentCardEmpty } from "./components/active-deployment-card/components/active-deployment-card-empty";
import { ActiveDeploymentCardSkeleton } from "./components/active-deployment-card/components/skeleton";
import { DeploymentDomainsCard } from "./components/deployment-domains-card";
import { ProjectContentWrapper } from "./components/project-content-wrapper";
import { Section, SectionHeader } from "./components/section";

export default function ProjectDetails() {
  const { isProjectLoading, project } = useProjectData();

  const liveDeploymentId = project?.liveDeploymentId;

  const domainsEmptyState = (
    <EmptySection
      title="No domains found"
      description="Your configured domains will appear here once they're set up and verified."
    />
  );

  return (
    <ProjectContentWrapper centered>
      {liveDeploymentId ? (
        <DeploymentLayoutProvider deploymentId={liveDeploymentId}>
          <DeploymentInfo title="Live Deployment" />
          <DeploymentDomainsCard
            domainFilter={(d) => d.sticky === "live"}
            emptyState={domainsEmptyState}
          />
        </DeploymentLayoutProvider>
      ) : (
        <>
          <Section>
            <SectionHeader
              icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
              title="Live Deployment"
            />
            {isProjectLoading ? <ActiveDeploymentCardSkeleton /> : <ActiveDeploymentCardEmpty />}
          </Section>
          <Section>
            <SectionHeader
              icon={<Earth iconSize="md-regular" className="text-gray-9" />}
              title="Domains"
            />
            {domainsEmptyState}
          </Section>
        </>
      )}
    </ProjectContentWrapper>
  );
}
