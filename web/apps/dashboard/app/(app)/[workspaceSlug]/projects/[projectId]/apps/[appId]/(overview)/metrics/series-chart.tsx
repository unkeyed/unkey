"use client";

import { type AreaChartPoint, AreaTimeseriesChart } from "@/components/charts/area-timeseries";
import type { ChartMarkers } from "@/components/charts/chart-markers";
import { formatBucketInterval } from "@/components/charts/format-interval";
import { NoDataPlaceholder } from "@/components/charts/no-data-placeholder";
import { DAY_MS, type Grid } from "@/lib/metrics/app-metrics";
import {
  type ChartState,
  type MetricUnit,
  type SeriesView,
  UNITS,
  formatValue,
} from "@/lib/metrics/views";
import { match } from "@unkey/match";
import type { ReactNode } from "react";
import type { ChartScale } from "./allocation";
import { useHoveredBucket, useSetHoveredBucket } from "./chart-hover";
import { legendReadout } from "./model";

const LEGEND_HEIGHT = 22;

type SeriesChartProps = {
  title: string;
  state: ChartState;
  series: Record<string, SeriesView>;
  scale: ChartScale;
  markers?: ChartMarkers;
  height: number;
  syncId: string;
  legend?: ReactNode;
  focusKey?: string | null;
  unavailableCopy?: string;
};

export function SeriesChart({
  title,
  state,
  series,
  scale,
  markers,
  height,
  syncId,
  legend,
  focusKey,
  unavailableCopy = `${title} metrics are unavailable right now`,
}: SeriesChartProps) {
  const frame = { legend, series, unit: scale.unit };
  return match(state)
    .with({ status: "empty" }, () => <NoDataPlaceholder height={height + LEGEND_HEIGHT} />)
    .with({ status: "failed" }, () => (
      <ChartFrame {...frame} data={null}>
        <div className="flex items-center justify-center text-xs text-gray-10" style={{ height }}>
          {unavailableCopy}
        </div>
      </ChartFrame>
    ))
    .with({ status: "loading" }, () => (
      <ChartFrame {...frame} data={null}>
        <AreaTimeseriesChart data={[]} isLoading config={series} height={height} />
      </ChartFrame>
    ))
    .with({ status: "ready" }, (ready) => (
      <ChartFrame {...frame} data={ready}>
        <GridChart
          chart={ready.chart}
          grid={ready.grid}
          series={series}
          scale={scale}
          markers={markers}
          height={height}
          syncId={syncId}
          focusKey={focusKey}
        />
      </ChartFrame>
    ))
    .exhaustive();
}

type LegendData = { chart: AreaChartPoint[]; grid: Grid } | null;

function ChartFrame({
  legend,
  data,
  series,
  unit,
  children,
}: {
  legend: ReactNode;
  data: LegendData;
  series: Record<string, SeriesView>;
  unit: MetricUnit;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      {legend ?? <LegendLine data={data} series={series} unit={unit} />}
      {children}
    </div>
  );
}

type GridChartProps = {
  chart: AreaChartPoint[];
  grid: Grid;
  series: Record<string, SeriesView>;
  scale: ChartScale;
  markers?: ChartMarkers;
  height: number;
  syncId: string;
  focusKey?: string | null;
};

function GridChart({
  chart,
  grid,
  series,
  scale,
  markers,
  height,
  syncId,
  focusKey,
}: GridChartProps) {
  const setHovered = useSetHoveredBucket();
  const format = UNITS[scale.unit];
  return (
    <AreaTimeseriesChart
      data={chart}
      config={series}
      height={height}
      fill="flat"
      showZeroLine
      showDateInTooltip={spansDays(grid)}
      formatTooltipValue={format.format}
      syncId={syncId}
      markers={markers}
      focusKey={focusKey}
      onActiveChange={(point) => setHovered(point?.originalTimestamp ?? null)}
      axis={{
        x: { domain: [grid.startMs, grid.endMs] },
        y: {
          floor: format.yFloor,
          max: scale.yMax ?? undefined,
          tickCount: scale.tickCount,
          formatTick: format.tick,
        },
      }}
    />
  );
}

function spansDays(grid: Grid): boolean {
  return grid.endMs - grid.startMs >= DAY_MS;
}

function LegendLine({
  data,
  series,
  unit,
}: {
  data: LegendData;
  series: Record<string, SeriesView>;
  unit: MetricUnit;
}) {
  const hoveredMs = useHoveredBucket();
  const keys = Object.keys(series);
  const readout = data && legendReadout(data.chart, keys, hoveredMs);
  return (
    <div className="flex min-h-4 flex-wrap items-center gap-x-3.5 gap-y-1 text-xs">
      {keys.map((key) => {
        const value = readout?.values[key] ?? null;
        return (
          <span key={key} className="flex items-center gap-1.5">
            <span className="size-2 rounded-full" style={{ backgroundColor: series[key].color }} />
            <span className="text-gray-10">{series[key].label}</span>
            <span className="font-medium tabular-nums text-gray-12">
              {value === null ? "—" : formatValue(unit, value)}
            </span>
          </span>
        );
      })}
      {data && readout && (
        <span className="ml-auto text-2xs tabular-nums text-gray-9">
          {formatBucketInterval(
            readout.atMs,
            readout.atMs + data.grid.bucketMs,
            spansDays(data.grid),
          )}
        </span>
      )}
    </div>
  );
}
