"use client";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Cloud, Earth, FolderCloud, Page2 } from "@unkey/icons";
import type { ReactNode } from "react";
import { ProjectContentWrapper } from "./components/project-content-wrapper";
import { ActiveDeploymentCard } from "./details/active-deployment-card";
import { DomainRow, DomainRowEmpty, DomainRowSkeleton } from "./details/domain-row";
import { EnvironmentVariablesSection } from "./details/env-variables-section";
import { useProject } from "./layout-provider";

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

  return (
    <ProjectContentWrapper centered>
      <Section>
        <SectionHeader
          icon={<Cloud size="md-regular" className="text-gray-9" />}
          title="Active Deployment"
        />
        <ActiveDeploymentCard deploymentId={project?.liveDeploymentId ?? null} />
      </Section>
      <Section>
        <SectionHeader icon={<Earth size="md-regular" className="text-gray-9" />} title="Domains" />
        <div>
          {isDomainsLoading ? (
            <>
              <DomainRowSkeleton />
              <DomainRowSkeleton />
            </>
          ) : domains?.length > 0 ? (
            domains.map((domain) => <DomainRow key={domain.id} domain={domain.domain} />)
          ) : (
            <DomainRowEmpty />
          )}
        </div>
      </Section>
      <Section>
        <SectionHeader
          icon={<FolderCloud size="md-regular" className="text-gray-9" />}
          title="Environment Variables"
        />
        <div>
          <EnvironmentVariablesSection
            icon={<Page2 className="text-gray-9" size="sm-medium" />}
            title="Production"
            projectId={projectId}
            environment="production"
          />
          <EnvironmentVariablesSection
            icon={<Page2 className="text-gray-9" size="sm-medium" />}
            title="Preview"
            projectId={projectId}
            environment="preview"
          />
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
