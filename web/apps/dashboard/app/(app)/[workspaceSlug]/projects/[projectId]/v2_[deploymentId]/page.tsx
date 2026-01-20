
"use client";

import { ReactNode } from "react";
import { ProjectContentWrapper } from "../components/project-content-wrapper";
import { Cloud } from "@unkey/icons";
import { ActiveDeploymentCard } from "../details/active-deployment-card";


const DEPLOYMENT_ID = "d_5VmWaBhBEn5jmAcZ"

export default function DeploymentOverview() {
  return (
    <ProjectContentWrapper centered>
      <Section>
        <SectionHeader
          icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
          title="Live Deployment"
        />
        <ActiveDeploymentCard deploymentId={DEPLOYMENT_ID} />
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
