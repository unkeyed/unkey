"use client";

import { Layers3 } from "@unkey/icons";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@unkey/ui";
import { useEffect, useState } from "react";
import { Section } from "../../../../../../components/section";
import { Card } from "../../../../../components/card";
import { useProjectData } from "../../../../../data-provider";
import { useDeployment } from "../../../layout-provider";
import { DeploymentBuildStepsTable } from "../table/deployment-build-steps-table";
import { DeploymentSentinelLogsTable } from "../table/deployment-sentinel-logs-table";

export function DeploymentLogsSection() {
  const { deploymentId } = useDeployment();
  const { getDeploymentById } = useProjectData();

  const deployment = getDeploymentById(deploymentId);
  const deploymentStatus = deployment?.status;

  // During build phase, default to "Build logs" and disable "Logs" tab
  const isReady = deploymentStatus !== "ready";

  const [tab, setTab] = useState(isReady ? "build-logs" : "requests");

  useEffect(() => {
    setTab(isReady ? "build-logs" : "requests");
  }, [isReady]);

  return (
    <Section>
      <Tabs value={tab} onValueChange={setTab}>
        <div className="flex items-center gap-2.5 py-1.5 px-2">
          <Layers3 iconSize="md-regular" className="text-gray-9" />
          <TabsList className="bg-gray-3">
            <TabsTrigger value="requests" className="text-accent-12 text-[13px]" disabled={isReady}>
              Requests
            </TabsTrigger>
            <TabsTrigger value="build-logs" className="text-accent-12 text-[13px]">
              Build logs
            </TabsTrigger>
          </TabsList>
        </div>
        <TabsContent value="requests">
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
