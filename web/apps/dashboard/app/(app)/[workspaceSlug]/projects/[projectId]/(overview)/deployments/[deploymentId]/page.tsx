"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import {
  Bolt,
  ChevronExpandY,
  Cloud,
  Grid,
  Harddrive,
  type IconProps,
  Layers2,
  Layers3,
  LayoutRight,
  TimeClock,
} from "@unkey/icons";
import {
  Button,
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useParams } from "next/navigation";
import type { ComponentType, ReactNode } from "react";
import { ActiveDeploymentCard } from "../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../components/deployment-status-badge";
import { InfoChip } from "../../../components/info-chip";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { Card } from "../../components/card";
import { useProject } from "../../layout-provider";
import { DeploymentSentinelLogsTable } from "./components/deployment-sentinel-logs-table";
import { DeploymentNetworkView } from "./network/deployment-network-view";
import { LogsTimeseriesBarChart } from "./network/unkey-flow/components/overlay/node-details-panel/components/chart";
import { generateRealisticChartData } from "./network/unkey-flow/components/overlay/node-details-panel/utils";

const baseConfig = {
  startTime: Date.now() - 24 * 60 * 60 * 1000 * 5,
  intervalMs: 60 * 60 * 1000,
};

export default function DeploymentOverview() {
  const params = useParams();
  const deploymentId = params?.deploymentId as string;

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
              <InfoChip icon={Bolt}>
                <div className="text-grayA-10 text-xs">
                  <span className="text-gray-12 font-medium">4</span> vCPUs
                </div>
              </InfoChip>
              <InfoChip icon={Grid}>
                <div className="text-grayA-10 text-xs">
                  <span className="text-gray-12 font-medium">4</span> GiB
                </div>
              </InfoChip>
              <InfoChip icon={Harddrive}>
                <div className="text-grayA-10 text-xs">
                  <span className="text-gray-12 font-medium">20</span> GB
                </div>
              </InfoChip>
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
          icon={<Layers2 iconSize="md-regular" className="text-gray-9" />}
          title="Network"
        />
        <div className="flex gap-2 flex-col">
          <Card className="rounded-[14px] flex justify-between flex-col overflow-hidden border-gray-4 h-[600px] gap-2">
            <DeploymentNetworkView projectId={projectId} liveDeploymentId={liveDeploymentId} />
          </Card>
          <div className="flex gap-2">
            <MetricCard
              icon={TimeClock}
              metricType="latency"
              currentValue={3.1}
              percentile="p50"
              chartData={{
                data: generateRealisticChartData({
                  count: 80,
                  baseValue: 3,
                  variance: 2,
                  trend: 0.01,
                  spikeProbability: 0.25,
                  dataKey: "latency",
                  ...baseConfig,
                }),
                dataKey: "latency",
              }}
            />
            <MetricCard
              icon={Bolt}
              metricType="cpu"
              currentValue={32}
              chartData={{
                data: generateRealisticChartData({
                  count: 80,
                  baseValue: 32,
                  variance: 10,
                  trend: 0.02,
                  spikeProbability: 0.15,
                  dataKey: "cpu",
                  ...baseConfig,
                }),
                dataKey: "cpu",
              }}
            />
            <MetricCard
              icon={Grid}
              metricType="memory"
              currentValue={24}
              secondaryValue={{ numeric: 1.62, unit: "gb" }}
              chartData={{
                data: generateRealisticChartData({
                  count: 80,
                  baseValue: 24,
                  variance: 8,
                  trend: 0.01,
                  spikeProbability: 0.1,
                  dataKey: "memory",
                  ...baseConfig,
                }),
                dataKey: "memory",
              }}
            />
            <MetricCard
              icon={Harddrive}
              metricType="storage"
              currentValue={32}
              secondaryValue={{ numeric: 72.3, unit: "mb" }}
              chartData={{
                data: generateRealisticChartData({
                  count: 80,
                  baseValue: 32,
                  variance: 5,
                  trend: 0.005,
                  spikeProbability: 0.05,
                  dataKey: "storage",
                  ...baseConfig,
                }),
                dataKey: "storage",
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
        <Card className="rounded-[14px] overflow-hidden border-gray-4 min-h-[600px] flex flex-col">
          <DeploymentSentinelLogsTable />
        </Card>
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

type MetricType = "latency" | "cpu" | "memory" | "storage";

type MetricConfig = {
  label: string;
  color: string;
  unit: string;
  percentiles?: string[];
};

const METRIC_CONFIGS: Record<MetricType, MetricConfig> = {
  latency: {
    label: "Latency",
    color: "hsl(var(--bronze-8))",
    unit: "ms",
    percentiles: ["p50", "p75", "p90", "p95", "p99"],
  },
  cpu: {
    label: "CPU",
    color: "hsl(var(--feature-8))",
    unit: "%",
  },
  memory: {
    label: "Memory",
    color: "hsl(var(--info-8))",
    unit: "%",
  },
  storage: {
    label: "Storage",
    color: "hsl(var(--cyan-8))",
    unit: "%",
  },
};

type MetricSelectProps = {
  label: string;
  value: string;
  options: string[];
  onValueChange?: (value: string) => void;
};

function MetricSelect({ label, value, options, onValueChange }: MetricSelectProps) {
  return (
    <Select defaultValue={value} onValueChange={onValueChange}>
      <SelectTrigger
        className="bg-transparent rounded-full flex items-center gap-1.5 border-0 h-auto !min-h-0 !p-0 focus:border-none focus:ring-0 hover:bg-grayA-2 transition-colors justify-normal "
        rightIcon={<ChevronExpandY className="text-accent-8 size-3.5" />}
      >
        <span className="text-gray-11 text-xs">{label}</span>
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="min-w-[80px]">
        {options.map((option) => (
          <SelectItem
            key={option}
            value={option}
            className="cursor-pointer hover:bg-grayA-3 data-[highlighted]:bg-grayA-2 font-mono font-medium text-sm"
          >
            {option}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

type MetricCardProps = {
  icon: ComponentType<IconProps>;
  metricType: MetricType;
  currentValue: number;
  secondaryValue?: {
    numeric: number;
    unit: string;
  };
  chartData: any;
  percentile?: string;
  onPercentileChange?: (value: string) => void;
};

function MetricCard({
  icon: Icon,
  metricType,
  currentValue,
  secondaryValue,
  chartData,
  percentile,
  onPercentileChange,
}: MetricCardProps) {
  const config = METRIC_CONFIGS[metricType];

  return (
    <div className="border border-gray-4 w-full h-28 rounded-xl flex flex-col">
      <div className="flex items-center w-full pt-[14px] px-[14px]">
        <div className="flex items-center w-full gap-1">
          <div className="flex items-center justify-center rounded-md bg-grayA-3 text-gray-12 size-5">
            <Icon iconSize="sm-regular" className="shrink-0" />
          </div>
          {config.percentiles && percentile ? (
            <MetricSelect
              label={config.label}
              value={percentile}
              options={config.percentiles}
              onValueChange={onPercentileChange}
            />
          ) : (
            <span className="text-gray-11 text-xs">{config.label}</span>
          )}
        </div>
        <div className="ml-auto tabular-nums">
          <span className="text-grayA-12 font-medium text-xs">{currentValue}</span>
          <span className="text-grayA-9 text-xs">{config.unit}</span>
          {secondaryValue && (
            <>
              <span className="text-grayA-12 font-medium text-xs ml-1">
                {secondaryValue.numeric}
              </span>
              <span className="text-grayA-9 text-xs">{secondaryValue.unit}</span>
            </>
          )}
        </div>
      </div>
      <div className="mt-1.5">
        <LogsTimeseriesBarChart
          chartContainerClassname="px-[14px] border-gray-4"
          data={chartData.data}
          config={{
            [chartData.dataKey]: {
              label: config.label,
              color: config.color,
            },
          }}
          height={48}
          isLoading={false}
          isError={false}
        />
      </div>
    </div>
  );
}
