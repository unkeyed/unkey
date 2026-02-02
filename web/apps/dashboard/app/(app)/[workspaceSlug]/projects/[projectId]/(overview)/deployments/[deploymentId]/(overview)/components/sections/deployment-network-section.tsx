"use client";

import { ChartActivity, Layers2, TimeClock } from "@unkey/icons";
import { useParams } from "next/navigation";
import { useState } from "react";
import { Section, SectionHeader } from "../../../../../../components/section";
import { Card } from "../../../../../components/card";
import { DeploymentNetworkView } from "../../../network/deployment-network-view";
import { useDeploymentLatency } from "../../hooks/use-deployment-latency";
import { useDeploymentRps } from "../../hooks/use-deployment-rps";
import { MetricCard } from "../metrics/metric-card";

export function DeploymentNetworkSection() {
  const [latencyPercentile, setLatencyPercentile] = useState<"p50" | "p75" | "p90" | "p95" | "p99">(
    "p50",
  );

  const params = useParams();
  const deploymentId = params?.deploymentId as string;
  const projectId = params?.projectId as string;

  const { currentRps, timeseries: rpsTimeseries } = useDeploymentRps(deploymentId);
  const { currentLatency, timeseries: latencyTimeseries } = useDeploymentLatency(
    deploymentId,
    latencyPercentile,
  );

  return (
    <Section>
      <SectionHeader
        icon={<Layers2 iconSize="md-regular" className="text-gray-9" />}
        title="Network"
      />
      <div className="flex gap-2 flex-col">
        <Card className="rounded-[14px] flex justify-between flex-col overflow-hidden border-gray-4 h-[600px] gap-2">
          <DeploymentNetworkView projectId={projectId} deploymentId={deploymentId} />
        </Card>
        <div className="flex gap-2">
          <MetricCard
            icon={TimeClock}
            metricType="latency"
            currentValue={currentLatency}
            percentile={latencyPercentile}
            onPercentileChange={(value) => setLatencyPercentile(value as typeof latencyPercentile)}
            chartData={{
              data: latencyTimeseries,
              dataKey: "y",
            }}
            timeWindow={{
              chart: "Last 6h",
            }}
          />
          <MetricCard
            icon={ChartActivity}
            metricType="rps"
            currentValue={currentRps}
            chartData={{
              data: rpsTimeseries,
              dataKey: "y",
            }}
            timeWindow={{
              chart: "Last 6h",
            }}
          />
        </div>
      </div>
    </Section>
  );
}
