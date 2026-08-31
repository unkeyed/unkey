"use client";

import {
  type AreaChartPoint,
  AreaTimeseriesChart,
  type ValueParts,
} from "@/components/charts/area-timeseries";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { ChartActivity, type IconProps, TimeClock, TriangleWarning2 } from "@unkey/icons";
import type { ComponentType } from "react";

export type LogdrainSeries = Array<{
  ts: number;
  successCount: number;
  transientErrorCount: number;
  permanentErrorCount: number;
  eventsDelivered: number;
  avgDurationMs: number;
}>;

type MetricChartProps = {
  icon: ComponentType<IconProps>;
  label: string;
  value: string;
  unit?: string;
  data: AreaChartPoint[];
  xDomain?: [number, number];
  dataKey: "eventsDelivered" | "errors" | "avgDurationMs";
  color: string;
  iconBg: string;
  iconText: string;
  loading: boolean;
  error: boolean;
  showZeroLine?: boolean;
  formatTooltipValue: (value: number) => ValueParts;
};

function MetricChart({
  icon: Icon,
  label,
  value,
  unit,
  data,
  xDomain,
  dataKey,
  color,
  iconBg,
  iconText,
  loading,
  error,
  showZeroLine,
  formatTooltipValue,
}: MetricChartProps) {
  return (
    <div className="flex w-full flex-col rounded-lg border border-gray-4 bg-grayA-1">
      <div className="flex w-full items-center gap-3 px-[14px] pt-3 pb-2">
        <div
          className={cn(
            "flex size-[22px] items-center justify-center rounded-md",
            iconBg,
            iconText,
          )}
        >
          <Icon iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-[13px] text-gray-12">{label}</span>
        <div className="ml-auto flex items-baseline gap-1">
          <span className="text-[13px] font-medium tabular-nums text-gray-12">
            {loading || error ? "‒" : value}
          </span>
          {unit && <span className="text-[11px] text-grayA-10">{unit}</span>}
        </div>
      </div>
      <div
        className="flex flex-col rounded-b-lg"
        style={{
          background: `linear-gradient(to top, color-mix(in srgb, ${error ? "hsl(var(--error-9))" : color} 6%, transparent), transparent)`,
        }}
      >
        <AreaTimeseriesChart
          chartContainerClassname="px-[14px]"
          data={data}
          config={{ [dataKey]: { label, color } }}
          height={50}
          isLoading={loading}
          isError={error}
          showZeroLine={showZeroLine}
          formatTooltipValue={formatTooltipValue}
          axis={xDomain ? { visible: false, x: { domain: xDomain }, y: { floor: 0 } } : null}
        />
        <span className="my-1 px-[14px] text-[10px] text-grayA-11">Past 24 hours</span>
      </div>
    </div>
  );
}

export function DeliveryOverview({
  series,
  loading,
  error,
}: {
  series?: LogdrainSeries;
  loading: boolean;
  error: boolean;
}) {
  const areaData: AreaChartPoint[] = (series ?? []).map((point) => ({
    originalTimestamp: new Date(point.ts).getTime(),
    eventsDelivered: Number(point.eventsDelivered),
    errors: Number(point.transientErrorCount) + Number(point.permanentErrorCount),
    avgDurationMs: Number(point.avgDurationMs),
  }));
  const firstPoint = areaData.at(0);
  const lastPoint = areaData.at(-1);
  const xDomain: [number, number] | undefined =
    firstPoint && lastPoint
      ? [firstPoint.originalTimestamp, lastPoint.originalTimestamp]
      : undefined;
  const totals = (series ?? []).reduce(
    (total, point) => {
      const failures = Number(point.transientErrorCount) + Number(point.permanentErrorCount);
      const attempts = Number(point.successCount) + failures;
      return {
        attempts: total.attempts + attempts,
        errors: total.errors + failures,
        events: total.events + Number(point.eventsDelivered),
        durationMs: total.durationMs + Number(point.avgDurationMs) * attempts,
      };
    },
    { attempts: 0, errors: 0, events: 0, durationMs: 0 },
  );
  const durationMs = totals.attempts === 0 ? null : totals.durationMs / totals.attempts;
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-sm font-medium text-accent-12">Delivery activity</h2>
      <div className="grid grid-cols-1 gap-2 md:grid-cols-3">
        <MetricChart
          icon={ChartActivity}
          label="Events delivered"
          value={formatNumber(totals.events)}
          data={areaData}
          xDomain={xDomain}
          dataKey="eventsDelivered"
          color="hsl(var(--activity))"
          iconBg="bg-info-3"
          iconText="text-info-11"
          loading={loading}
          error={error}
          showZeroLine
          formatTooltipValue={(value) => ({ value: value.toLocaleString() })}
        />
        <MetricChart
          icon={TriangleWarning2}
          label="Failed attempts"
          value={formatNumber(totals.errors)}
          data={areaData}
          xDomain={xDomain}
          dataKey="errors"
          color="hsl(var(--error-9))"
          iconBg="bg-error-3"
          iconText="text-error-11"
          loading={loading}
          error={error}
          showZeroLine
          formatTooltipValue={(value) => ({ value: value.toLocaleString() })}
        />
        <MetricChart
          icon={TimeClock}
          label="Average attempt time"
          value={durationMs === null ? "—" : Math.round(durationMs).toLocaleString()}
          unit="ms"
          data={areaData}
          xDomain={xDomain}
          dataKey="avgDurationMs"
          color="hsl(var(--bronze-8))"
          iconBg="bg-bronze-3"
          iconText="text-bronze-11"
          loading={loading}
          error={error}
          showZeroLine
          formatTooltipValue={(value) => ({
            value: Math.round(value).toLocaleString(),
            unit: "ms",
          })}
        />
      </div>
    </section>
  );
}
