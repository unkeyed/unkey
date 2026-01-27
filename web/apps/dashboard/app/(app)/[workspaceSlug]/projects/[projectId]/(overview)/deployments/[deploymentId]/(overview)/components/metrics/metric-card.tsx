import type { IconProps } from "@unkey/icons";
import type { ComponentType } from "react";
import { LogsTimeseriesBarChart } from "../../../network/unkey-flow/components/overlay/node-details-panel/components/chart";
import { MetricSelect } from "./metric-select";
import type { TimeseriesData } from "@/components/logs/overview-charts/types";
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

type MetricCardProps = {
  icon: ComponentType<IconProps>;
  metricType: MetricType;
  currentValue: number;
  secondaryValue?: {
    numeric: number;
    unit: string;
  };
  chartData: { data?: TimeseriesData[], dataKey: string };
  percentile?: string;
  onPercentileChange?: (value: string) => void;
};

export function MetricCard({
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
