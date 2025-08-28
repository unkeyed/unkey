"use client";
import { Cloud, Earth, FolderCloud, Page2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { ActiveDeploymentCard, type DeploymentStatus } from "./details/active-deployment-card";
import { CollapsibleRow } from "./details/collapsible-row";
import { DomainRow } from "./details/domain-row";
import { ProjectLayout } from "./project-layout";

const DEPLOYMENT_DATA = {
  version: "v_alpha001",
  description: "Add auth routes + logging",
  status: "active" as DeploymentStatus,
  author: {
    name: "Oz",
    avatar: "https://avatars.githubusercontent.com/u/138932600?s=48&v=4",
  },
  createdAt: "a day ago",
  branch: "main",
  commit: "e5f6a7b",
  image: "unkey:latest",
};

const DOMAINS = [
  {
    domain: "api.gateway.com",
    status: "success" as const,
    tags: ["https", "primary"],
  },
  {
    domain: "dev.gateway.com",
    status: "error" as const,
    tags: ["https", "primary"],
  },
  {
    domain: "staging.gateway.com",
    status: "success" as const,
    tags: ["https", "primary"],
  },
];

const ENVS = ["Production", "Preview"];

export default function ProjectDetails({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <ProjectLayout projectId={projectId}>
      {({ isDetailsOpen }) => (
        <div
          className={cn(
            "flex justify-center transition-all duration-300 ease-in-out",
            isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]",
          )}
        >
          <div className="max-w-[960px] flex flex-col w-full mt-4 gap-5">
            <Section>
              <SectionHeader
                icon={<Cloud size="md-regular" className="text-gray-9" />}
                title="Active Deployment"
              />
              <ActiveDeploymentCard {...DEPLOYMENT_DATA} />
            </Section>

            <Section>
              <SectionHeader
                icon={<Earth size="md-regular" className="text-gray-9" />}
                title="Domains"
              />
              <div>
                {DOMAINS.map((domain) => (
                  <DomainRow key={domain.domain} {...domain} />
                ))}
              </div>
            </Section>

            <Section>
              <SectionHeader
                icon={<FolderCloud size="md-regular" className="text-gray-9" />}
                title="Environment Variables"
              />
              <div>
                {ENVS.map((env) => (
                  <CollapsibleRow
                    key={env}
                    icon={<Page2 className="text-gray-9" size="sm-medium" />}
                    title={env}
                  />
                ))}
              </div>
            </Section>
          </div>
        </div>
      )}
    </ProjectLayout>
  );
}

type SectionHeaderProps = {
  icon: ReactNode;
  title: string;
};

function SectionHeader({ icon, title }: SectionHeaderProps) {
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
