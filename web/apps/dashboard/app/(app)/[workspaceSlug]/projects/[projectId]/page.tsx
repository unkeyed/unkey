"use client";
import { Cloud, Earth } from "@unkey/icons";
import { EmptySection } from "./(overview)/components/empty-section";
import { useProjectData } from "./(overview)/data-provider";
import { DeploymentInfo } from "./(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-info";
import { DeploymentLayoutProvider } from "./(overview)/deployments/[deploymentId]/layout-provider";
import { DomainRow, DomainRowSkeleton } from "./(overview)/details/domain-row";
import { ActiveDeploymentCardEmpty } from "./components/active-deployment-card/components/active-deployment-card-empty";
import { ActiveDeploymentCardSkeleton } from "./components/active-deployment-card/components/skeleton";
import { ProjectContentWrapper } from "./components/project-content-wrapper";
import { Section, SectionHeader } from "./components/section";

export default function ProjectDetails() {
  const { getDomainsForDeployment, isDomainsLoading, isProjectLoading, project } =
    useProjectData();

  const liveDeploymentId = project?.liveDeploymentId;

  // Get domains for live deployment
  const domains = liveDeploymentId
    ? getDomainsForDeployment(liveDeploymentId).filter((d) => d.sticky === "live")
    : [];

  return (
    <ProjectContentWrapper centered>
      {liveDeploymentId ? (
        <DeploymentLayoutProvider deploymentId={liveDeploymentId}>
          <DeploymentInfo title="Live Deployment" />
        </DeploymentLayoutProvider>
      ) : (
        <Section>
          <SectionHeader
            icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
            title="Live Deployment"
          />
          {isProjectLoading ? <ActiveDeploymentCardSkeleton /> : <ActiveDeploymentCardEmpty />}
        </Section>
      )}
      <Section>
        <SectionHeader
          icon={<Earth iconSize="md-regular" className="text-gray-9" />}
          title="Domains"
        />
        <div>
          {isDomainsLoading ? (
            <>
              <DomainRowSkeleton />
              <DomainRowSkeleton />
            </>
          ) : domains.length > 0 ? (
            domains.map((domain) => (
              <DomainRow key={domain.id} domain={domain.fullyQualifiedDomainName} />
            ))
          ) : (
            <EmptySection
              title="No domains found"
              description="Your configured domains will appear here once they're set up and verified."
            />
          )}
        </div>
      </Section>
    </ProjectContentWrapper>
  );
}
