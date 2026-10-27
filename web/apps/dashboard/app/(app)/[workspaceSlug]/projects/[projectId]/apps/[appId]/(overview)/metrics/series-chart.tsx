"use client";

import { type AreaChartPoint, AreaTimeseriesChart } from "@/components/charts/area-timeseries";
import type { ChartMarkers } from "@/components/charts/chart-markers";
import { NoDataPlaceholder } from "@/components/charts/no-data-placeholder";
import { type Grid, spansDays } from "@/lib/metrics/app-metrics";
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

const LEGEND_HEIGHT = 33;

type SeriesChartProps = {
  title: string;
  state: ChartState;
  series: Record<string, SeriesView>;
  scale: ChartScale;
  markers?: ChartMarkers;
  tooltipFooter?: (point: AreaChartPoint) => ReactNode;
  height: number;
  syncId: string;
  legend?: ReactNode;
  focusKey?: string | null;
  connectNulls?: boolean;
  unavailableCopy?: string;
};

export function SeriesChart({
  title,
  state,
  series,
  scale,
  markers,
  tooltipFooter,
  height,
  syncId,
  legend,
  focusKey,
  connectNulls,
  unavailableCopy = `${title} metrics are unavailable right now`,
}: SeriesChartProps) {
  const frame = { legend, series, unit: scale.unit };
  return match(state)
    .with({ status: "empty" }, () => <NoDataPlaceholder height={height + LEGEND_HEIGHT} />)
    .with({ status: "failed" }, () => (
      <ChartFrame {...frame} chart={null}>
        <div className="flex items-center justify-center text-xs text-gray-10" style={{ height }}>
          {unavailableCopy}
        </div>
      </ChartFrame>
    ))
    .with({ status: "loading" }, () => (
      <ChartFrame {...frame} chart={null}>
        <AreaTimeseriesChart
          data={[]}
          isLoading
          config={series}
          height={height}
          fill="flat"
          formatTooltipValue={UNITS[scale.unit].format}
        />
      </ChartFrame>
    ))
    .with({ status: "ready" }, (ready) => (
      <ChartFrame {...frame} chart={ready.chart}>
        <GridChart
          chart={ready.chart}
          grid={ready.grid}
          series={series}
          scale={scale}
          markers={markers}
          tooltipFooter={tooltipFooter}
          height={height}
          syncId={syncId}
          focusKey={focusKey}
          connectNulls={connectNulls}
        />
      </ChartFrame>
    ))
    .exhaustive();
}

function ChartFrame({
  legend,
  chart,
  series,
  unit,
  children,
}: {
  legend: ReactNode;
  chart: AreaChartPoint[] | null;
  series: Record<string, SeriesView>;
  unit: MetricUnit;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3">
      {children}
      <div className="-mx-4 -mb-3 flex flex-wrap border-t border-grayA-4 text-xs">
        {legend ?? <SeriesLegendCells chart={chart} series={series} unit={unit} />}
      </div>
    </div>
  );
}

type GridChartProps = {
  chart: AreaChartPoint[];
  grid: Grid;
  series: Record<string, SeriesView>;
  scale: ChartScale;
  markers?: ChartMarkers;
  tooltipFooter?: (point: AreaChartPoint) => ReactNode;
  height: number;
  syncId: string;
  focusKey?: string | null;
  connectNulls?: boolean;
};

function GridChart({
  chart,
  grid,
  series,
  scale,
  markers,
  tooltipFooter,
  height,
  syncId,
  focusKey,
  connectNulls,
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
      tooltipFooter={tooltipFooter}
      focusKey={focusKey}
      connectNulls={connectNulls}
      onActiveChange={(point) => setHovered(point?.originalTimestamp ?? null)}
      axis={{
        x: { domain: [grid.startMs, grid.endMs - grid.bucketMs] },
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

function SeriesLegendCells({
  chart,
  series,
  unit,
}: {
  chart: AreaChartPoint[] | null;
  series: Record<string, SeriesView>;
  unit: MetricUnit;
}) {
  const hoveredMs = useHoveredBucket();
  const keys = Object.keys(series);
  const readout = chart && legendReadout(chart, keys, hoveredMs);
  return keys.map((key) => {
    const value = readout?.[key] ?? null;
    return (
      <LegendCell key={key} color={series[key].color} label={series[key].label}>
        <span className="font-medium tabular-nums text-gray-12">
          {value === null ? "—" : formatValue(unit, value)}
        </span>
      </LegendCell>
    );
  });
}

export function LegendCell({
  color,
  label,
  children,
}: {
  color: string;
  label: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="flex min-w-36 flex-1 items-center justify-between gap-3 border-l border-grayA-4 px-4 py-2 first:border-l-0">
      <span className="flex items-center gap-2 text-gray-11">
        <span className="size-2.5 shrink-0 rounded-xs" style={{ backgroundColor: color }} />
        {label}
      </span>
      {children}
    </div>
  );
}
