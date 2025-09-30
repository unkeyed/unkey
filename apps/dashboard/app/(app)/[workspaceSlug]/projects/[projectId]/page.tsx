"use client";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Cloud, Earth, FolderCloud, Page2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { ActiveDeploymentCard } from "./details/active-deployment-card";
import {
  DomainRow,
  DomainRowEmpty,
  DomainRowSkeleton,
} from "./details/domain-row";
import { EnvironmentVariablesSection } from "./details/env-variables-section";
import { useProjectLayout } from "./layout-provider";

export default function ProjectDetails() {
  const { isDetailsOpen, projectId, collections } = useProjectLayout();

  const projects = useLiveQuery((q) =>
    q
      .from({ project: collection.projects })
      .where(({ project }) => eq(project.id, projectId))
  );

  const project = projects.data.at(0);
  const { data: domains, isLoading: isDomainsLoading } = useLiveQuery(
    (q) =>
      q
        .from({ domain: collections.domains })
        .where(({ domain }) =>
          eq(domain.deploymentId, project?.liveDeploymentId)
        ),
    [project?.liveDeploymentId]
  );

  return (
    <div
      className={cn(
        "flex justify-center transition-all duration-300 ease-in-out pb-20 px-8",
        isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]"
      )}
    >
      <div className="max-w-[960px] flex flex-col w-full mt-4 gap-5">
        <Section>
          <SectionHeader
            icon={<Cloud iconsize="md-regular" className="text-gray-9" />}
            title="Active Deployment"
          />
          <ActiveDeploymentCard
            deploymentId={project?.liveDeploymentId ?? null}
          />
        </Section>
        <Section>
          <SectionHeader
            icon={<Earth iconsize="md-medium" className="text-gray-9" />}
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
                <DomainRow key={domain.id} domain={domain.domain} />
              ))
            ) : (
              <DomainRowEmpty />
            )}
          </div>
        </Section>
        <Section>
          <SectionHeader
            icon={<FolderCloud iconsize="md-medium" className="text-gray-9" />}
            title="Environment Variables"
          />
          <div>
            <EnvironmentVariablesSection
              icon={<Page2 className="text-gray-9" iconsize="sm-medium" />}
              title="Production"
              projectId={projectId}
              environment="production"
            />
            <EnvironmentVariablesSection
              icon={<Page2 className="text-gray-9" iconsize="sm-medium" />}
              title="Preview"
              projectId={projectId}
              environment="preview"
            />
          </div>
        </Section>
      </div>
    </div>
  );
}

function SectionHeader({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="flex items-center gap-2.5 py-1.5 px-2">
      {icon}
      <div className="text-accent-12 font-medium text-[13px] leading-4">
        {title}
      </div>
    </div>
  );
}

function Section({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-1">{children}</div>;
}
