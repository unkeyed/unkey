"use client";

import {
  bytesToMib,
  formatMemoryParts,
  formatTooltipPercent,
} from "@/lib/utils/deployment-formatters";
import type { PERCENTILE_VALUES } from "@unkey/clickhouse/src/sentinel";
import { ChartActivity, Layers2, Microchip, Ram, TimeClock } from "@unkey/icons";
import { useMemo, useState } from "react";
import { Section, SectionHeader } from "../../../../../../components/section";
import { Card } from "../../../../../components/card";
import { useDeployment } from "../../../layout-provider";
import { DeploymentNetworkView } from "../../../network/deployment-network-view";
import { useDeploymentCpu } from "../../hooks/use-deployment-cpu";
import { useDeploymentLatency } from "../../hooks/use-deployment-latency";
import { useDeploymentMemory } from "../../hooks/use-deployment-memory";
import { useDeploymentRps } from "../../hooks/use-deployment-rps";
import { MetricCard, formatMetricParts } from "../metrics/metric-card";

export function DeploymentNetworkSection() {
  const [latencyPercentile, setLatencyPercentile] = useState<keyof typeof PERCENTILE_VALUES>("p50");

  const { deployment } = useDeployment();

  const sixHourDomain = useMemo((): [number, number] => {
    const now = Date.now();
    return [now - 6 * 60 * 60 * 1000, now];
  }, []);

  const {
    currentRps,
    timeseries: rpsTimeseries,
    isLoading: isRpsLoading,
    isError: isRpsError,
  } = useDeploymentRps(deployment.id);
  const {
    currentLatency,
    timeseries: latencyTimeseries,
    isLoading: isLatencyLoading,
    isError: isLatencyError,
  } = useDeploymentLatency(deployment.id, latencyPercentile);

  const {
    cpuPercent,
    allocatedMillicores,
    timeseries: cpuTimeseries,
    isLoading: isCpuLoading,
    isError: isCpuError,
  } = useDeploymentCpu(deployment.id);

  const {
    memoryPercent,
    memoryDisplay,
    allocatedBytes,
    timeseries: memoryTimeseries,
    isLoading: isMemoryLoading,
    isError: isMemoryError,
  } = useDeploymentMemory(deployment.id);

  return (
    <Section>
      <SectionHeader
        icon={<Layers2 iconSize="md-regular" className="text-gray-9" />}
        title="Network"
      />
      <div className="flex gap-2 flex-col">
        <Card className="flex justify-between flex-col overflow-hidden h-[600px] gap-2">
          <DeploymentNetworkView showNodeDetails />
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
            xAxisDomain={sixHourDomain}
            timeWindow={{
              chart: "Last 6h",
            }}
            isLoading={isLatencyLoading}
            isError={isLatencyError}
            formatTooltipValue={(v) => {
              const p = formatMetricParts("latency", v, "ms");
              return { value: p.value, unit: p.unit };
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
            isLoading={isRpsLoading}
            isError={isRpsError}
            formatTooltipValue={(v) => ({ value: v.toFixed(1), unit: "req/s" })}
          />
          <MetricCard
            icon={Microchip}
            metricType="cpu"
            currentValue={cpuPercent}
            chartData={{
              data: cpuTimeseries,
              dataKey: "y",
            }}
            xAxisDomain={sixHourDomain}
            timeWindow={{
              chart: "Last 6h",
            }}
            isLoading={isCpuLoading}
            isError={isCpuError}
            formatTooltipValue={(v) => {
              if (allocatedMillicores <= 0) {
                return { value: `${Math.round(v)}`, unit: "m" };
              }
              const pct = formatTooltipPercent((v / allocatedMillicores) * 100);
              return { value: pct, hint: `${Math.round(v)}m` };
            }}
          />
          <MetricCard
            icon={Ram}
            metricType="memory"
            currentValue={memoryPercent}
            secondaryValue={{
              numeric: memoryDisplay.value,
              unit: memoryDisplay.unit,
            }}
            chartData={{
              data: memoryTimeseries,
              dataKey: "y",
            }}
            xAxisDomain={sixHourDomain}
            timeWindow={{
              chart: "Last 6h",
            }}
            isLoading={isMemoryLoading}
            isError={isMemoryError}
            formatTooltipValue={(v) => {
              const parts = formatMemoryParts(bytesToMib(v));
              if (allocatedBytes <= 0) {
                return { value: parts.value, unit: parts.unit };
              }
              return {
                value: parts.value,
                unit: parts.unit,
                hint: `(${formatTooltipPercent((v / allocatedBytes) * 100)})`,
              };
            }}
          />
        </div>
      </div>
    </Section>
  );
}
