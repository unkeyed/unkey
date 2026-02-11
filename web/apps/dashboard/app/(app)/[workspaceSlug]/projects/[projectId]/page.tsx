"use client";
import { Cloud, Earth, FolderCloud, Link4, Page2 } from "@unkey/icons";
import { useProjectData } from "./(overview)/data-provider";
import { DeploymentLogsContent } from "./(overview)/details/active-deployment-card-logs/components/deployment-logs-content";
import { DeploymentLogsTrigger } from "./(overview)/details/active-deployment-card-logs/components/deployment-logs-trigger";
import { DeploymentLogsProvider } from "./(overview)/details/active-deployment-card-logs/providers/deployment-logs-provider";
import { CustomDomainsSection } from "./(overview)/details/custom-domains-section";
import { DomainRow, EmptySection, DomainRowSkeleton } from "./(overview)/details/domain-row";
import { EnvironmentVariablesSection } from "./(overview)/details/env-variables-section";
import { ActiveDeploymentCard } from "./components/active-deployment-card";
import { DeploymentStatusBadge } from "./components/deployment-status-badge";
import { ProjectContentWrapper } from "./components/project-content-wrapper";
import { Section, SectionHeader } from "./components/section";

export default function ProjectDetails() {
  const { projectId, getDomainsForDeployment, isDomainsLoading, getDeploymentById, project, environments } =
    useProjectData();

  const liveDeploymentId = project?.liveDeploymentId;

  // Get domains for live deployment
  const domains = liveDeploymentId ? getDomainsForDeployment(liveDeploymentId) : [];

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
            trailingContent={<DeploymentLogsTrigger />}
            expandableContent={
              project?.liveDeploymentId ? (
                <DeploymentLogsContent
                  projectId={projectId}
                  deploymentId={project?.liveDeploymentId}
                />
              ) : null
            }
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
      <Section>
        <SectionHeader
          icon={<Link4 iconSize="md-regular" className="text-gray-9" />}
          title="Custom Domains"
        />
        <CustomDomainsSection
          environments={environments.map((env) => ({ id: env.id, slug: env.slug }))}
        />
      </Section>
      <Section>
        <SectionHeader
          icon={<FolderCloud iconSize="md-regular" className="text-gray-9" />}
          title="Environment Variables"
        />
        <div>
          {environments.map((env) => (
            <EnvironmentVariablesSection
              key={env.id}
              icon={<Page2 iconSize="sm-medium" className="text-gray-9" />}
              title={env.slug}
              environment={env.slug}
            />
          ))}
          {environments.length === 0 && (
            <div className="px-4 py-8 text-center text-gray-9 text-sm">
              No environments configured
            </div>
          )}
        </div>
      </Section>
    </ProjectContentWrapper>
  );
}
