"use client";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Cloud, Earth, FolderCloud, Page2 } from "@unkey/icons";
import type { ReactNode } from "react";
import { ProjectContentWrapper } from "./components/project-content-wrapper";
import { DomainRow, DomainRowEmpty, DomainRowSkeleton } from "./details/domain-row";
import { useProject } from "./layout-provider";
import { ActiveDeploymentCard } from "./components/active-deployment-card";
import { DeploymentStatusBadge } from "./components/deployment-status-badge";
import { EnvironmentVariablesSection } from "./details/env-variables-section";

export default function ProjectDetails() {
  const { projectId, collections } = useProject();

  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
  );

  const project = projects.data.at(0);
  const { data: domains, isLoading: isDomainsLoading } = useLiveQuery(
    (q) =>
      q
        .from({ domain: collections.domains })
        .where(({ domain }) => eq(domain.deploymentId, project?.liveDeploymentId)),
    [project?.liveDeploymentId],
  );

  const { data: environments } = useLiveQuery((q) => q.from({ env: collections.environments }));

  const deployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, project?.liveDeploymentId)),
    [project?.liveDeploymentId],
  );
  const deploymentStatus = deployment.data.at(0)?.status;


  return (
    <ProjectContentWrapper centered>
      <Section>
        <SectionHeader
          icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
          title="Live Deployment"
        />
        <ActiveDeploymentCard deploymentId={project?.liveDeploymentId ?? null}
          statusBadge={
            <DeploymentStatusBadge
              status={deploymentStatus}
              className="text-successA-11 font-medium"
            />
          }
        />
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
          ) : domains?.length > 0 ? (
            domains.map((domain) => (
              <DomainRow key={domain.id} domain={domain.fullyQualifiedDomainName} />
            ))
          ) : (
            <DomainRowEmpty />
          )}
        </div>
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

function SectionHeader({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="flex items-center gap-2.5 py-1.5 px-2">
      {icon}
      <div className="text-accent-12 font-medium text-[13px] leading-4">{title}</div>
    </div>
  );
}

function Section({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-1">{children}</div>;
}
