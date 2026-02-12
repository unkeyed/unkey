"use client";

import { Section } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/section";
import { Card } from "../../../../../components/card";
import { DeploymentBuildStepsTable } from "../table/deployment-build-steps-table";

export function DeploymentLogsSection() {
  return (
    <Section>
      <Card className="rounded-[14px] overflow-hidden border-gray-4 flex flex-col h-full">
        <DeploymentBuildStepsTable />
      </Card>
    </Section>
  );
}
