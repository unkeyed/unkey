import type { TimeseriesData } from "@/components/logs/overview-charts/types";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { IconProps } from "@unkey/icons";
import type { ComponentType } from "react";
import { LogsTimeseriesBarChart } from "../../../network/unkey-flow/components/overlay/node-details-panel/components/chart";
import { MetricSelect } from "./metric-select";

type MetricType = "latency" | "rps";

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
  rps: {
    label: "RPS",
    color: "hsl(var(--feature-8))",
    unit: "req/s",
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
  chartData: { data?: TimeseriesData[]; dataKey: string };
  percentile?: string;
  onPercentileChange?: (value: string) => void;
  timeWindow?: {
    chart: string;
  };
};

export function MetricCard({
  icon: Icon,
  metricType,
  currentValue,
  secondaryValue,
  chartData,
  percentile,
  onPercentileChange,
  timeWindow,
}: MetricCardProps) {
  const config = METRIC_CONFIGS[metricType];

  return (
    <div className="border border-gray-4 w-full rounded-xl flex flex-col">
      <div className="flex items-start w-full pt-[10px] px-[14px]">
        <div className="flex items-center w-full gap-2">
          <div className="flex items-center justify-center rounded-md bg-grayA-3 text-gray-12 size-5">
            <Icon iconSize="sm-regular" className="shrink-0" />
          </div>
          <div className="flex flex-col gap-0.5">
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
        </div>
        <div className="ml-auto flex flex-col">
          <div className="flex gap-0.5 items-center">
            {metricType === "latency" ? (
              <span className="text-grayA-12 font-medium text-xs">
                {formatLatency(currentValue)}
              </span>
            ) : (
              <>
                <span className="text-grayA-12 font-medium text-xs">{currentValue}</span>
                <span className="text-grayA-9 text-xs"> {config.unit}</span>
              </>
            )}
          </div>
          {secondaryValue && (
            <>
              <span className="text-grayA-12 font-medium text-xs ml-1">
                {secondaryValue.numeric}
              </span>
              <span className="text-grayA-9 text-xs"> {secondaryValue.unit}</span>
            </>
          )}
        </div>
      </div>
      <div className="mt-6 flex flex-col">
        <LogsTimeseriesBarChart
          chartContainerClassname="px-[14px] border-gray-4"
          data={chartData.data}
          config={{
            [chartData.dataKey]: {
              label: config.label,
              color: config.color,
            },
          }}
          height={50}
          isLoading={false}
          isError={false}
        />
        {timeWindow?.chart && (
          <span className="text-grayA-9 text-[10px] px-[14px] my-1">{timeWindow.chart}</span>
        )}
      </div>
    </div>
  );
}
