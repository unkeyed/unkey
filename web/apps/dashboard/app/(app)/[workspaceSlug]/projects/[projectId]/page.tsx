"use client";
import { Cloud, Earth } from "@unkey/icons";
import { EmptySection } from "./(overview)/components/empty-section";
import { useProjectData } from "./(overview)/data-provider";
import { DeploymentLogsProvider } from "./(overview)/details/active-deployment-card-logs/providers/deployment-logs-provider";
import { DomainRow, DomainRowSkeleton } from "./(overview)/details/domain-row";
import { ActiveDeploymentCard } from "./components/active-deployment-card";
import { DeploymentStatusBadge } from "./components/deployment-status-badge";
import { ProjectContentWrapper } from "./components/project-content-wrapper";
import { Section, SectionHeader } from "./components/section";

export default function ProjectDetails() {
  const { getDomainsForDeployment, isDomainsLoading, getDeploymentById, project } =
    useProjectData();

  const liveDeploymentId = project?.liveDeploymentId;

  // Get domains for live deployment
  const domains = liveDeploymentId
    ? getDomainsForDeployment(liveDeploymentId).filter((d) => d.sticky === "live")
    : [];

  // Get deployment from provider
  const deploymentStatus = liveDeploymentId
    ? getDeploymentById(liveDeploymentId)?.status
    : undefined;

  return (
    <ProjectContentWrapper centered>
      <Section>
        <SectionHeader
          icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
          title="Live Deployment"
        />
        <DeploymentLogsProvider>
          <ActiveDeploymentCard
            deploymentId={project?.liveDeploymentId ?? null}
            statusBadge={<DeploymentStatusBadge status={deploymentStatus} />}
          />
        </DeploymentLogsProvider>{" "}
      </Section>
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
