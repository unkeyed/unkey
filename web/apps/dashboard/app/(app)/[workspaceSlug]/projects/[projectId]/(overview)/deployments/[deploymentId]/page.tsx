"use client";

import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import {
  Bolt,
  ChartActivity,
  Cloud,
  Earth,
  Grid,
  Harddrive,
  Layers2,
  Layers3,
  LayoutRight,
  TimeClock,
} from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { useParams } from "next/navigation";
import { useState } from "react";
import { ActiveDeploymentCard } from "../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../components/deployment-status-badge";
import { DisabledWrapper } from "../../../components/disabled-wrapper";
import { InfoChip } from "../../../components/info-chip";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { Section, SectionHeader } from "../../../components/section";
import { Card } from "../../components/card";
import { DomainRow, DomainRowEmpty, DomainRowSkeleton } from "../../details/domain-row";
import { useProject } from "../../layout-provider";
import { MetricCard } from "./(overview)/components/metrics/metric-card";
import { DeploymentSentinelLogsTable } from "./(overview)/components/table/deployment-sentinel-logs-table";
import { DeploymentNetworkView } from "./network/deployment-network-view";

export default function DeploymentOverview() {
  const params = useParams();
  const deploymentId = params?.deploymentId as string;
  const [latencyPercentile, setLatencyPercentile] = useState<"p50" | "p75" | "p90" | "p95" | "p99">(
    "p50",
  );

  const { collections, setIsDetailsOpen, isDetailsOpen, projectId, liveDeploymentId } =
    useProject();
  const deployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [deploymentId],
  );
  const deploymentStatus = deployment.data.at(0)?.status;

  const { data: domains, isLoading: isDomainsLoading } = useLiveQuery(
    (q) =>
      q
        .from({ domain: collections.domains })
        .where(({ domain }) => eq(domain.deploymentId, deploymentId)),
    [deploymentId],
  );

  // RPS data queries
  const { data: currentRps } = trpc.deploy.metrics.getDeploymentRps.useQuery({
    deploymentId,
  });

  const { data: rpsTimeseries } = trpc.deploy.metrics.getDeploymentRpsTimeseries.useQuery({
    deploymentId,
  });

  // Latency data queries
  const { data: currentLatency } = trpc.deploy.metrics.getDeploymentLatency.useQuery({
    deploymentId,
    percentile: latencyPercentile,
  });

  const { data: latencyTimeseries } = trpc.deploy.metrics.getDeploymentLatencyTimeseries.useQuery({
    deploymentId,
    percentile: latencyPercentile,
  });

  const chartData =
    rpsTimeseries?.map((d) => ({
      originalTimestamp: d.x,
      y: d.y,
      total: d.y,
    })) ?? [];

  const latencyChartData =
    latencyTimeseries?.map((d) => ({
      originalTimestamp: d.x,
      y: d.y,
      total: d.y,
    })) ?? [];

  return (
    <ProjectContentWrapper centered>
      <Section>
        <SectionHeader
          icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
          title="Deployment"
        />
        <ActiveDeploymentCard
          deploymentId={deploymentId}
          trailingContent={
            <div className="flex gap-1.5 items-center">
              <DisabledWrapper
                tooltipContent="Resource metrics coming soon"
                className="flex gap-1.5 items-center"
              >
                <InfoChip icon={Bolt}>
                  <div className="text-grayA-10 text-xs">
                    <span className="text-gray-12 font-medium">—</span> vCPUs
                  </div>
                </InfoChip>
                <InfoChip icon={Grid}>
                  <div className="text-grayA-10 text-xs">
                    <span className="text-gray-12 font-medium">—</span> GiB
                  </div>
                </InfoChip>
                <InfoChip icon={Harddrive}>
                  <div className="text-grayA-10 text-xs">
                    <span className="text-gray-12 font-medium">—</span> GB
                  </div>
                </InfoChip>
              </DisabledWrapper>
              <div className="gap-1 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
                <div className="border rounded-[10px] border-grayA-3 size-4 bg-grayA-3 flex items-center justify-center">
                  <img src={"/images/flags/us.svg"} alt="us-flag" className="size-4" />
                </div>
                <div className="border rounded-[10px] border-grayA-3 size-4 bg-grayA-3 flex items-center justify-center">
                  <img src={"/images/flags/de.svg"} alt="de-flag" className="size-4" />
                </div>
                <div className="border rounded-[10px] border-grayA-3 size-4 bg-grayA-3 flex items-center justify-center">
                  <img src={"/images/flags/in.svg"} alt="in-flag" className="size-4" />
                </div>
              </div>
              <InfoTooltip asChild content="Show deployment details">
                <Button
                  variant="ghost"
                  className="[&_svg]:size-3 size-3 rounded-sm"
                  size="icon"
                  onClick={() => setIsDetailsOpen(!isDetailsOpen)}
                >
                  <LayoutRight iconSize="sm-regular" className="text-gray-10" />
                </Button>
              </InfoTooltip>
            </div>
          }
          statusBadge={
            <DeploymentStatusBadge
              status={deploymentStatus}
              className="text-successA-11 font-medium"
            />
          }
        />
      </Section>
      <Section>
        <SectionHeader
          icon={<Earth iconSize="md-regular" className="text-gray-9" />}
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
              <DomainRow key={domain.id} domain={domain.fullyQualifiedDomainName} />
            ))
          ) : (
            <DomainRowEmpty />
          )}
        </div>
      </Section>
      <Section>
        <SectionHeader
          icon={<Layers2 iconSize="md-regular" className="text-gray-9" />}
          title="Network"
        />
        <div className="flex gap-2 flex-col">
          <Card className="rounded-[14px] flex justify-between flex-col overflow-hidden border-gray-4 h-[600px] gap-2">
            <DeploymentNetworkView projectId={projectId} deploymentId={liveDeploymentId} />
          </Card>
          <div className="flex gap-2">
            <MetricCard
              icon={TimeClock}
              metricType="latency"
              currentValue={currentLatency?.latency ?? 0}
              percentile={latencyPercentile}
              onPercentileChange={(value) =>
                setLatencyPercentile(value as typeof latencyPercentile)
              }
              chartData={{
                data: latencyChartData,
                dataKey: "y",
              }}
              timeWindow={{
                chart: "Last 6h",
              }}
            />
            <MetricCard
              icon={ChartActivity}
              metricType="rps"
              currentValue={currentRps?.avg_rps ?? 0}
              chartData={{
                data: chartData,
                dataKey: "y",
              }}
              timeWindow={{
                chart: "Last 6h",
              }}
            />
          </div>
        </div>
      </Section>

      <Section>
        <SectionHeader
          icon={<Layers3 iconSize="md-regular" className="text-gray-9" />}
          title="Logs"
        />
        <Card className="rounded-[14px] overflow-hidden border-gray-4 flex flex-col h-full">
          <DeploymentSentinelLogsTable />
        </Card>
      </Section>
    </ProjectContentWrapper>
  );
}
