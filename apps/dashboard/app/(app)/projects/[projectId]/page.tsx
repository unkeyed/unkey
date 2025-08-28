// Updated project details component
"use client";
import { Cloud, Earth, FolderCloud, Page2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { ActiveDeploymentCard, type DeploymentStatus } from "./details/active-deployment-card";
import { DomainRow } from "./details/domain-row";
import { EnvironmentVariablesSection } from "./details/env-variables-section";
import { useProjectLayout } from "./layout-provider";

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

// Mock environment variables data
const PRODUCTION_VARS = [
  {
    id: "1",
    key: "DATABASE_URL",
    value: "postgresql://user:pass@prod.db.com:5432/app",
    isSecret: true,
  },
  {
    id: "2",
    key: "API_KEY",
    value: "sk_prod_1234567890abcdef",
    isSecret: true,
  },
  { id: "3", key: "NODE_ENV", value: "production", isSecret: false },
  {
    id: "4",
    key: "REDIS_URL",
    value: "redis://prod.redis.com:6379",
    isSecret: true,
  },
  { id: "5", key: "LOG_LEVEL", value: "info", isSecret: false },
];

const PREVIEW_VARS = [
  {
    id: "6",
    key: "DATABASE_URL",
    value: "postgresql://user:pass@staging.db.com:5432/app",
    isSecret: true,
  },
  {
    id: "7",
    key: "API_KEY",
    value: "sk_test_abcdef1234567890",
    isSecret: true,
  },
  { id: "8", key: "NODE_ENV", value: "development", isSecret: false },
];

export default function ProjectDetails() {
  const { isDetailsOpen } = useProjectLayout();

  return (
    <div
      className={cn(
        "flex justify-center transition-all duration-300 ease-in-out pb-20",
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
            <EnvironmentVariablesSection
              icon={<Page2 className="text-gray-9" size="sm-medium" />}
              title="Production"
              initialVars={PRODUCTION_VARS}
              initialOpen
            />
            <EnvironmentVariablesSection
              icon={<Page2 className="text-gray-9" size="sm-medium" />}
              title="Preview"
              initialVars={PREVIEW_VARS}
            />
          </div>
        </Section>
      </div>
    </div>
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
