"use client";
import { Cloud, Earth, FolderCloud, Page2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { ActiveDeploymentCard } from "./details/active-deployment-card";
import { DomainRow } from "./details/domain-row";
import { EnvironmentVariablesSection } from "./details/env-variables-section";
import { useProjectLayout } from "./layout-provider";

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

export default function ProjectDetails() {
  const { isDetailsOpen, projectId } = useProjectLayout();

  return (
    <div
      className={cn(
        "flex justify-center transition-all duration-300 ease-in-out pb-20 px-8",
        isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]",
      )}
    >
      <div className="max-w-[960px] flex flex-col w-full mt-4 gap-5">
        <Section>
          <SectionHeader
            icon={<Cloud size="md-regular" className="text-gray-9" />}
            title="Active Deployment"
          />
          <ActiveDeploymentCard />
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
      </div>
    </div>
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
