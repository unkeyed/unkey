"use client";

import { ChartEmpty } from "@/components/logs/chart/chart-states";
import { type ChartConfig, ChartContainer, ChartTooltip } from "@/components/ui/chart";
import { formatTimestamp } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useId, useState } from "react";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { type AreaFill, FILL_STYLES, Gradients } from "./area-fill";
import { resolveXAxisDomain } from "./chart-domain";
import {
  ChartMarkerOverlay,
  type ChartMarkers,
  MarkerLayer,
  activeMarkerSpan,
} from "./chart-markers";
import { ChartError } from "./components/chart-error";
import { ChartWaveLoading } from "./components/chart-wave-loading";
import { formatBucketInterval } from "./format-interval";

export type AreaChartPoint = { originalTimestamp: number } & {
  [k: string]: number | undefined;
};

function isAreaChartPoint(value: unknown): value is AreaChartPoint {
  return (
    typeof value === "object" &&
    value !== null &&
    "originalTimestamp" in value &&
    typeof value.originalTimestamp === "number"
  );
}

function activePointFromState(state: unknown, data: AreaChartPoint[]): AreaChartPoint | null {
  if (typeof state !== "object" || state === null) {
    return null;
  }
  const raw =
    "activeTooltipIndex" in state
      ? state.activeTooltipIndex
      : "activeIndex" in state
        ? state.activeIndex
        : undefined;
  let idx = Number.NaN;
  if (typeof raw === "number") {
    idx = raw;
  } else if (typeof raw === "string") {
    idx = Number.parseInt(raw, 10);
  }
  if (!Number.isInteger(idx) || idx < 0 || idx >= data.length) {
    return null;
  }
  const candidate = data[idx];
  return isAreaChartPoint(candidate) ? candidate : null;
}

const Y_GUTTER_PX = 36;
const DIMMED_OPACITY = 0.18;

export type ValueParts = { value: string; unit?: string; hint?: string };

export type AreaTimeseriesAxisOptions = {
  visible?: boolean;
  x?: {
    domain?: [number, number];
    contractOnSparseData?: boolean;
    utc?: boolean;
  };
  y?: {
    floor?: number;
    max?: number;
    tickCount?: number;
    formatTick?: (value: number) => string;
  };
};

function useSyncedScrub(syncId: string | undefined) {
  const [inside, setInside] = useState(false);
  return {
    showDetail: !syncId || inside,
    containerProps: syncId
      ? { onPointerEnter: () => setInside(true), onPointerLeave: () => setInside(false) }
      : {},
  };
}

type Props = {
  data: AreaChartPoint[];
  config: ChartConfig;
  height?: number;
  isLoading?: boolean;
  isError?: boolean;
  chartContainerClassname?: string;
  showDateInTooltip?: boolean;
  formatTooltipValue: (value: number) => ValueParts;
  axis?: AreaTimeseriesAxisOptions | null;
  fill: AreaFill;
  fillColors?: Record<string, string>;
  onActiveChange?: (point: AreaChartPoint | null) => void;
  hideTooltip?: boolean;
  showZeroLine?: boolean;
  syncId?: string;
  markers?: ChartMarkers;
  focusKey?: string | null;
};

function yTop(dataMax: number, floor: number, max: number | undefined, zeroLine: boolean) {
  if (max !== undefined) {
    return Math.max(max, dataMax * 1.1);
  }
  return zeroLine ? Math.max(floor, 1) : Math.max(dataMax, floor) * 1.1;
}

export function AreaTimeseriesChart({
  data,
  config,
  height = 140,
  isLoading,
  isError,
  chartContainerClassname,
  showDateInTooltip,
  formatTooltipValue,
  axis,
  fill,
  fillColors,
  onActiveChange,
  hideTooltip,
  showZeroLine,
  syncId,
  markers,
  focusKey = null,
}: Props) {
  const handleActive = onActiveChange
    ? (state: unknown) => onActiveChange(activePointFromState(state, data))
    : undefined;
  const chartId = useId().replace(/:/g, "");
  const [shouldAnimate, setShouldAnimate] = useState(true);
  const scrub = useSyncedScrub(syncId);
  useEffect(() => {
    const timer = setTimeout(() => setShouldAnimate(false), 600);
    return () => clearTimeout(timer);
  }, []);

  if (isError) {
    return <ChartError height={height} />;
  }

  const configKeys = Object.keys(config);
  const firstKey = configKeys[0];
  const sectionColor = config[firstKey]?.color;

  if (isLoading) {
    return <ChartWaveLoading height={height} color={sectionColor} />;
  }
  const values = data.flatMap((point) => configKeys.map((key) => point[key]));
  const reported = values.filter((value) => value !== undefined);
  const hasPositiveValue = reported.some((value) => value > 0);
  const isAllZero = reported.length > 0 && reported.every((value) => value === 0);
  const renderZeroLine = Boolean(showZeroLine && isAllZero);
  if (!data.length || (!hasPositiveValue && !renderZeroLine)) {
    return (
      <ChartEmpty variant="wave" color={sectionColor} height={height} message="No activity yet" />
    );
  }

  const dataMax = data.reduce(
    (m, p) => configKeys.reduce((mm, k) => Math.max(mm, Number(p[k]) || 0), m),
    0,
  );
  const yAxisFloor = axis?.y?.floor ?? 0;
  const formatYTick = axis?.y?.formatTick ?? String;
  const top = yTop(dataMax, yAxisFloor, axis?.y?.max, renderZeroLine);
  const tickCount = Math.max(2, axis?.y?.tickCount ?? 4);
  const yTicks = Array.from({ length: tickCount }, (_, i) => (i * top) / (tickCount - 1));
  const yDomain: [number, number] = [0, top];

  let firstNonZeroTs: number | undefined;
  let lastNonZeroTs: number | undefined;
  for (const p of data) {
    if (configKeys.some((k) => Number(p[k]) > 0)) {
      firstNonZeroTs ??= p.originalTimestamp;
      lastNonZeroTs = p.originalTimestamp;
    }
  }
  const { effectiveDomain, spanMs } = resolveXAxisDomain({
    xAxisDomain: axis?.x?.domain,
    contractOnSparseData: axis?.x?.contractOnSparseData,
    firstNonZeroTs,
    lastNonZeroTs,
  });
  const xTickFormatter = (v: number) => formatXAxisTick(v, spanMs, axis?.x?.utc);
  const showAxes = axis !== null && axis?.visible !== false;

  const xTicks = effectiveDomain
    ? [0, 1, 2, 3].map((i) =>
        Math.round(effectiveDomain[0] + (i * (effectiveDomain[1] - effectiveDomain[0])) / 3),
      )
    : undefined;

  const margin = {
    top: showAxes || markers ? 16 : 4,
    right: showAxes ? 8 : 0,
    bottom: 0,
    left: 0,
  };
  const markerDomain: [number, number] = effectiveDomain ?? [
    data[0].originalTimestamp,
    data[data.length - 1].originalTimestamp,
  ];
  const markerSpan = markers
    ? activeMarkerSpan(markers.items, markers.activeId, markerDomain)
    : null;

  const style = FILL_STYLES[fill];
  const emphasis = (key: string) => (focusKey === null || key === focusKey ? 1 : DIMMED_OPACITY);
  const { showDetail } = scrub;

  const chart = (
    <ChartContainer
      config={config}
      className={cn("!flex-col aspect-auto w-full", style.containerClass, chartContainerClassname)}
      style={{ height, width: "100%" }}
      {...scrub.containerProps}
    >
      <AreaChart
        data={data}
        onMouseMove={handleActive}
        onMouseLeave={onActiveChange ? () => onActiveChange(null) : undefined}
        syncId={syncId}
        margin={margin}
      >
        {style.gradient && <Gradients chartId={chartId} config={config} fillColors={fillColors} />}
        {showAxes && <CartesianGrid vertical={false} {...style.grid} />}
        <XAxis
          dataKey="originalTimestamp"
          type="number"
          domain={effectiveDomain ?? ["dataMin", "dataMax"]}
          allowDataOverflow={Boolean(effectiveDomain)}
          scale="time"
          tickFormatter={xTickFormatter}
          tick={showAxes ? { fill: "var(--color-gray-10)", fontSize: 10 } : false}
          tickLine={false}
          axisLine={false}
          ticks={xTicks}
          minTickGap={48}
          hide={!showAxes}
        />
        <YAxis
          width={showAxes ? Y_GUTTER_PX : 0}
          tickLine={false}
          axisLine={false}
          tickFormatter={formatYTick}
          tick={showAxes ? { fill: "var(--color-gray-10)", fontSize: 10 } : false}
          ticks={yTicks}
          domain={yDomain}
          hide={!showAxes}
        />
        {markers && <MarkerLayer markers={markers} span={markerSpan} />}
        <ChartTooltip
          allowEscapeViewBox={{ x: false, y: true }}
          wrapperStyle={{ zIndex: 1000, pointerEvents: "none" }}
          cursor={style.cursor}
          content={({ active, payload }) => {
            if (hideTooltip || !showDetail || !active || !payload?.length) {
              return null;
            }
            const point: unknown = payload[0]?.payload;
            if (!isAreaChartPoint(point)) {
              return null;
            }
            const labelText = formatCompactInterval(
              point.originalTimestamp,
              data,
              showDateInTooltip,
            );
            const rows = configKeys
              .map((key) => ({ key, value: point[key] }))
              .sort((a, b) => (b.value ?? -1) - (a.value ?? -1));
            return (
              <div
                role="tooltip"
                className="grid w-max max-w-[300px] animate-in items-start gap-1.5 rounded-xl bg-raised/80 px-3 py-2.5 text-xs shadow-floating backdrop-blur-md duration-150 fade-in-0 zoom-in-95 select-none"
              >
                <div className="font-medium text-2xs text-gray-11">{labelText}</div>
                <div className="grid gap-1">
                  {rows.map(({ key, value }) => {
                    const itemConfig = config[key];
                    const parts =
                      value === undefined ? { value: "No data" } : formatTooltipValue(value);
                    return (
                      <div key={key} className="flex items-center gap-2">
                        <div
                          className="shrink-0 rounded-xs h-2 w-2"
                          style={{ backgroundColor: itemConfig?.color }}
                        />
                        <span className="text-gray-12">{itemConfig?.label ?? key}</span>
                        <span className="font-mono tabular-nums text-gray-12 ml-auto">
                          {parts.value}
                          {parts.unit && ` ${parts.unit}`}
                          {parts.hint && (
                            <span className="text-grayA-9 ml-1 font-normal">{parts.hint}</span>
                          )}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          }}
        />
        {configKeys.map((key) => (
          <Area
            key={key}
            dataKey={key}
            type="monotone"
            stroke={config[key].color}
            strokeWidth={key === focusKey ? 2 : 1.5}
            strokeOpacity={emphasis(key)}
            fill={style.gradient ? `url(#${chartId}-${key})` : config[key].color}
            fillOpacity={style.fillOpacity * emphasis(key)}
            isAnimationActive={shouldAnimate}
            animationDuration={500}
            animationEasing="ease-out"
            dot={false}
            activeDot={(props: { cx?: number; cy?: number }) =>
              showDetail ? style.dot(props.cx ?? 0, props.cy ?? 0, config[key].color) : <g />
            }
          />
        ))}
      </AreaChart>
    </ChartContainer>
  );

  if (!markers) {
    return chart;
  }
  return (
    <div className="relative w-full">
      {chart}
      <ChartMarkerOverlay
        markers={markers}
        domain={markerDomain}
        insetLeft={margin.left + (showAxes ? Y_GUTTER_PX : 0)}
        insetRight={margin.right}
        bandHeight={margin.top}
      />
    </div>
  );
}

const TWO_DAYS_MS = 2 * 24 * 60 * 60 * 1000;
function formatXAxisTick(v: number, spanMs: number, utc?: boolean): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  return formatTimestamp(v, spanMs >= TWO_DAYS_MS ? "MMM d" : "h:mm a", utc ? "UTC" : undefined);
}

function formatCompactInterval(
  startTs: number | undefined,
  data: AreaChartPoint[],
  withDate?: boolean,
): string {
  if (typeof startTs !== "number" || !Number.isFinite(startTs)) {
    return "";
  }
  const idx = data.findIndex((p) => p.originalTimestamp === startTs);
  const nextTs = idx >= 0 ? data[idx + 1]?.originalTimestamp : undefined;
  return formatBucketInterval(startTs, typeof nextTs === "number" ? nextTs : undefined, withDate);
}
