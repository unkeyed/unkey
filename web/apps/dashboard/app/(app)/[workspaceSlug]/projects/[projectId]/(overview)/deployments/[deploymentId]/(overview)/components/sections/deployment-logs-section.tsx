"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { Layers3 } from "@unkey/icons";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@unkey/ui";
import { useParams } from "next/navigation";
import { Section } from "../../../../../../components/section";
import { Card } from "../../../../../components/card";
import { useProject } from "../../../../../layout-provider";
import { DeploymentBuildStepsTable } from "../table/deployment-build-steps-table";
import { DeploymentSentinelLogsTable } from "../table/deployment-sentinel-logs-table";

export function DeploymentLogsSection() {
  const params = useParams();
  const deploymentId = params?.deploymentId as string;

  const { collections } = useProject();
  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [deploymentId],
  );

  const deployment = data.at(0);
  const deploymentStatus = deployment?.status;

  // During build phase, default to "Build logs" and disable "Logs" tab
  const isBuildPhase = deploymentStatus === "building";
  const defaultTab = isBuildPhase ? "build-logs" : "sentinel";

  return (
    <Section>
      <Tabs defaultValue={defaultTab}>
        <div className="flex items-center gap-2.5 py-1.5 px-2">
          <Layers3 iconSize="md-regular" className="text-gray-9" />
          <TabsList className="bg-gray-3">
            <TabsTrigger
              value="sentinel"
              className="text-accent-12 text-[13px]"
              disabled={isBuildPhase}
            >
              Logs
            </TabsTrigger>
            <TabsTrigger value="build-logs" className="text-accent-12 text-[13px]">
              Build logs
            </TabsTrigger>
          </TabsList>
        </div>
        <TabsContent value="sentinel">
          <Card className="rounded-[14px] overflow-hidden border-gray-4 flex flex-col h-full">
            <DeploymentSentinelLogsTable />
          </Card>
        </TabsContent>
        <TabsContent value="build-logs">
          <Card className="rounded-[14px] overflow-hidden border-gray-4 flex flex-col h-full">
            <DeploymentBuildStepsTable />
          </Card>
        </TabsContent>
      </Tabs>
    </Section>
  );
}
