"use client";

import { Layers3 } from "@unkey/icons";
import { Section, SectionHeader } from "../../../../../../components/section";
import { Card } from "../../../../../components/card";
import { DeploymentSentinelLogsTable } from "../table/deployment-sentinel-logs-table";

export function DeploymentLogsSection() {
  return (
    <Section>
      <SectionHeader
        icon={<Layers3 iconSize="md-regular" className="text-gray-9" />}
        title="Logs"
      />
      <Card className="rounded-[14px] overflow-hidden border-gray-4 flex flex-col h-full">
        <DeploymentSentinelLogsTable />
      </Card>
    </Section>
  );
}
