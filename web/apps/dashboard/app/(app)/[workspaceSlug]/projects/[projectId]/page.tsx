"use client";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Cloud, Earth, FolderCloud, Link4, Page2 } from "@unkey/icons";
import { useProjectData } from "./(overview)/data-provider";
import { DeploymentLogsContent } from "./(overview)/details/active-deployment-card-logs/components/deployment-logs-content";
import { DeploymentLogsTrigger } from "./(overview)/details/active-deployment-card-logs/components/deployment-logs-trigger";
import { DeploymentLogsProvider } from "./(overview)/details/active-deployment-card-logs/providers/deployment-logs-provider";
import { CustomDomainsSection } from "./(overview)/details/custom-domains-section";
import { DomainRow, DomainRowEmpty, DomainRowSkeleton } from "./(overview)/details/domain-row";
import { EnvironmentVariablesSection } from "./(overview)/details/env-variables-section";
import { useProject } from "./(overview)/layout-provider";
import { ActiveDeploymentCard } from "./components/active-deployment-card";
import { DeploymentStatusBadge } from "./components/deployment-status-badge";
import { ProjectContentWrapper } from "./components/project-content-wrapper";
import { Section, SectionHeader } from "./components/section";

export default function ProjectDetails() {
  const { projectId, liveDeploymentId } = useProject();
  const { getDomainsForDeployment, isDomainsLoading, getDeploymentById } = useProjectData();

  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
  );

  const project = projects.data.at(0);

  // Get domains for live deployment
  const domains = liveDeploymentId ? getDomainsForDeployment(liveDeploymentId) : [];

  const { data: environments } = useLiveQuery(
    (q) =>
      q.from({ env: collection.environments }).where(({ env }) => eq(env.projectId, projectId)),
    [projectId],
  );

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
            <DomainRowEmpty
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
          projectId={projectId}
          environments={environments?.map((env) => ({ id: env.id, slug: env.slug })) ?? []}
        />
      </Section>
      <Section>
        <SectionHeader
          icon={<FolderCloud iconSize="md-regular" className="text-gray-9" />}
          title="Environment Variables"
        />
        <div>
          {environments?.map((env) => (
            <EnvironmentVariablesSection
              key={env.id}
              icon={<Page2 iconSize="sm-medium" className="text-gray-9" />}
              title={env.slug}
              projectId={projectId}
              environment={env.slug}
            />
          ))}
          {environments?.length === 0 && (
            <div className="px-4 py-8 text-center text-gray-9 text-sm">
              No environments configured
            </div>
          )}
        </div>
      </Section>
    </ProjectContentWrapper>
  );
}
