"use client";

import { type AreaChartPoint, AreaTimeseriesChart } from "@/components/charts/area-timeseries";
import type { ChartConfig } from "@/components/ui/chart";
import { formatNumber } from "@/lib/fmt";
import { Loading, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import { useMemo, useState } from "react";
import { formatStamp } from "./g-pulse";
import { useProductionCard } from "./production-card-context";

const BLUE = "var(--color-activity)";
const BLUE_FILL = "var(--color-info-3)";
const ERROR = "var(--color-error-9)";
const ERROR_FILL = "var(--color-error-3)";

const CHART_CONFIG: ChartConfig = {
  total: { label: "Requests/s", color: BLUE },
  errors: { label: "Errors/s", color: ERROR },
};

// Hoisted so its identity is stable: a fresh object each render resets recharts' cursor.
const FILL_COLORS = { total: BLUE_FILL, errors: ERROR_FILL };

const formatCountTooltip = (value: number) => ({ value: `${Math.round(value)}`, unit: "req" });
const COUNT_TICK_FORMATTER = new Intl.NumberFormat("en", {
  notation: "compact",
  maximumSignificantDigits: 1,
});

function formatCountYTick(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  return COUNT_TICK_FORMATTER.format(v);
}

function LegendStat({
  color,
  label,
  value,
  alert,
}: {
  color: string;
  label: string;
  value: string;
  alert?: boolean;
}) {
  return (
    <span className="flex items-center gap-1.5 whitespace-nowrap tabular-nums">
      <span className="size-2 shrink-0 rounded-full" style={{ backgroundColor: color }} />
      <span className="text-gray-9">{label}</span>
      <span className={cn("font-medium", alert ? "text-error-11" : "text-gray-12")}>{value}</span>
    </span>
  );
}

export function BuildInProgressChart() {
  return (
    <div className="flex flex-col gap-2 p-4 md:border-r">
      <div className="flex flex-col gap-1">
        <Skeleton className="h-6 w-20" />
        <Skeleton className="h-3 w-28" />
      </div>
      <Skeleton className="h-[120px] w-full rounded-md" />
      <div className="flex items-center gap-2 text-sm text-gray-9">
        <Loading size={16} className="text-gray-12" />
        Waiting for build to finish…
      </div>
    </div>
  );
}

export function ProductionCardChart() {
  const { pulse, isChartLoading, isChartError } = useProductionCard();
  const [active, setActive] = useState<AreaChartPoint | null>(null);

  const reqValue = active ? Number(active.total) || 0 : pulse.requestsCurrent;
  const errValue = active ? Number(active.errors) || 0 : pulse.errorsCurrent;
  const stampTs = active ? active.originalTimestamp : pulse.latestTimestamp;
  const xDomain = useMemo<[number, number] | undefined>(
    () =>
      pulse.series.length > 1
        ? [
            pulse.series[0].originalTimestamp,
            pulse.series[pulse.series.length - 1].originalTimestamp,
          ]
        : undefined,
    [pulse.series],
  );

  return (
    <div className="flex flex-col gap-2 p-4 md:border-r">
      <div className="flex items-start justify-between gap-2">
        <div className="flex flex-col">
          <span className="text-2xl font-semibold text-gray-12 tabular-nums leading-tight">
            {formatNumber(pulse.cumulative)}
          </span>
          <span className="text-sm text-gray-9">requests {pulse.windowLabel}</span>
        </div>
        <span className="text-sm tabular-nums text-gray-9">
          {formatStamp(stampTs, pulse.windowKey, active !== null)}
        </span>
      </div>
      <AreaTimeseriesChart
        data={pulse.series}
        config={CHART_CONFIG}
        fillColors={FILL_COLORS}
        paleFill
        height={120}
        axis={{
          x: { domain: xDomain, utc: true },
          y: { floor: 0, formatTick: formatCountYTick },
        }}
        isLoading={isChartLoading}
        isError={isChartError}
        formatTooltipValue={formatCountTooltip}
        hideTooltip
        onActiveChange={setActive}
      />
      <div className="flex items-center gap-4 text-sm">
        <LegendStat color={BLUE} label="Requests/s" value={formatNumber(reqValue)} />
        <LegendStat
          color={ERROR}
          label="Errors/s"
          value={formatNumber(errValue)}
          alert={errValue > 0}
        />
      </div>
    </div>
  );
}
