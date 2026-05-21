import type { IconProps } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ComponentType } from "react";
import { LogsTimeseriesBarChart } from "../../../network/unkey-flow/components/overlay/node-details-panel/components/chart";
import {
  type AreaChartPoint,
  AreaTimeseriesChart,
  type ValueParts,
} from "../../../network/unkey-flow/components/overlay/node-details-panel/components/chart/area-timeseries-chart";
import { MetricSelect } from "./metric-select";

type MetricType = "latency" | "rps" | "cpu" | "memory";

type ChartVariant = "bar" | "area";

type MetricConfig = {
  label: string;
  color: string;
  iconBg: string;
  iconText: string;
  unit: string;
  chartVariant: ChartVariant;
  percentiles?: string[];
};

const METRIC_CONFIGS: Record<MetricType, MetricConfig> = {
  latency: {
    label: "Latency",
    color: "hsl(var(--bronze-8))",
    iconBg: "bg-bronze-3",
    iconText: "text-bronze-11",
    unit: "ms",
    chartVariant: "area",
    percentiles: ["p50", "p75", "p90", "p95", "p99"],
  },
  rps: {
    label: "RPS",
    color: "hsl(var(--accent-8))",
    iconBg: "bg-accent-3",
    iconText: "text-accent-11",
    unit: "req/s",
    chartVariant: "bar",
  },
  cpu: {
    label: "CPU",
    color: "hsl(var(--feature-8))",
    iconBg: "bg-feature-3",
    iconText: "text-feature-11",
    unit: "%",
    chartVariant: "area",
  },
  memory: {
    label: "Memory",
    color: "hsl(var(--info-8))",
    iconBg: "bg-info-3",
    iconText: "text-info-11",
    unit: "%",
    chartVariant: "area",
  },
};

type MetricCardProps = {
  icon: ComponentType<IconProps>;
  metricType: MetricType;
  currentValue: number;
  secondaryValue?: {
    numeric: number | string;
    unit: string;
  };
  chartData: { data?: AreaChartPoint[]; dataKey: string };
  percentile?: string;
  onPercentileChange?: (value: string) => void;
  timeWindow?: {
    chart: string;
  };
  xAxisDomain?: [number, number];
  isLoading?: boolean;
  isError?: boolean;
  formatTooltipValue?: (value: number) => ValueParts;
};

export function MetricCard({
  icon: Icon,
  metricType,
  currentValue,
  secondaryValue,
  chartData,
  percentile,
  onPercentileChange,
  xAxisDomain,
  timeWindow,
  isLoading = false,
  isError = false,
  formatTooltipValue,
}: MetricCardProps) {
  const config = METRIC_CONFIGS[metricType];
  const parts = formatMetricParts(metricType, currentValue, config.unit);
  const noData = isError || isLoading;
  const valueText = noData ? "‒" : parts.value;
  const secondaryText = noData ? "‒" : secondaryValue?.numeric;
  const gradientColor = isError ? "hsl(var(--error-9))" : config.color;

  return (
    <div className="border border-gray-4 bg-grayA-1 w-full rounded-[14px] flex flex-col">
      <div className="flex items-center gap-3 w-full px-[14px] pt-[12px] pb-[8px]">
        <div
          className={cn(
            "flex items-center justify-center rounded-md size-[22px]",
            config.iconBg,
            config.iconText,
          )}
        >
          <Icon iconSize="sm-regular" className="shrink-0" />
        </div>
        <div className="flex flex-col">
          {config.percentiles && percentile ? (
            <MetricSelect
              label={config.label}
              value={percentile}
              options={config.percentiles}
              onValueChange={onPercentileChange}
            />
          ) : (
            <span className="text-gray-12 text-[13px]">{config.label}</span>
          )}
        </div>
        <div className="ml-auto flex items-baseline gap-1">
          <span className="text-gray-12 font-medium text-[13px] tabular-nums">{valueText}</span>
          <span className="text-grayA-10 text-[11px]">{parts.unit}</span>
          {secondaryValue && (
            <>
              <span className="text-grayA-9 text-[11px]">/</span>
              <span className="text-gray-12 font-medium text-[12px] tabular-nums">
                {secondaryText}
              </span>
              <span className="text-grayA-10 text-[11px]">{secondaryValue.unit}</span>
            </>
          )}
        </div>
      </div>
      <div
        className="flex flex-col rounded-b-[14px]"
        style={{
          background: `linear-gradient(to top, color-mix(in srgb, ${gradientColor} 6%, transparent), transparent)`,
        }}
      >
        {config.chartVariant === "area" ? (
          <AreaTimeseriesChart
            chartContainerClassname="px-[14px]"
            data={chartData.data ?? []}
            config={{
              [chartData.dataKey]: {
                label: config.label,
                color: config.color,
              },
            }}
            height={50}
            isLoading={isLoading}
            isError={isError}
            formatTooltipValue={formatTooltipValue}
            axisFloor={0}
            xAxisDomain={xAxisDomain}
            hideAxes
          />
        ) : (
          <LogsTimeseriesBarChart
            chartContainerClassname="px-[14px] border-gray-3"
            data={chartData.data}
            config={{
              [chartData.dataKey]: {
                label: config.label,
                color: config.color,
              },
            }}
            height={50}
            isLoading={isLoading}
            isError={isError}
            formatTooltipValue={formatTooltipValue}
          />
        )}
        {timeWindow?.chart && (
          <span className="text-grayA-11 text-[10px] px-[14px] my-1">{timeWindow.chart}</span>
        )}
      </div>
    </div>
  );
}

export function formatMetricParts(
  type: MetricType,
  value: number,
  defaultUnit: string,
): { value: string; unit: string } {
  if (type === "cpu" || type === "memory") {
    return { value: `${Math.round(value)}`, unit: "%" };
  }
  if (type === "latency") {
    if (value < 1000) {
      return { value: `${Math.round(value * 10) / 10}`, unit: "ms" };
    }
    const seconds = value / 1000;
    if (seconds < 60) {
      return { value: seconds.toFixed(1), unit: "s" };
    }
    const minutes = seconds / 60;
    if (minutes < 60) {
      return { value: minutes.toFixed(1), unit: "m" };
    }
    const hours = minutes / 60;
    return { value: hours.toFixed(1), unit: "h" };
  }
  return { value: `${value}`, unit: defaultUnit };
}
