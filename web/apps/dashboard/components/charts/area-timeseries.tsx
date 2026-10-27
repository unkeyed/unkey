"use client";

import { ChartEmpty } from "@/components/logs/chart/chart-states";
import { type ChartConfig, ChartContainer, ChartTooltip } from "@/components/ui/chart";
import { useMeasuredWidth } from "@/hooks/use-measured-width";
import { cn } from "@unkey/ui/src/lib/utils";
import { type ReactNode, useEffect, useId, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  XAxis,
  type XAxisTickContentProps,
  YAxis,
  type YAxisTickContentProps,
} from "recharts";
import { type AreaFill, FILL_STYLES, Gradients } from "./area-fill";
import { AxisEdgeLabel } from "./axis-edge-label";
import { type AxisEdge, axisEdges } from "./axis-edges";
import { resolveXAxisDomain } from "./chart-domain";
import {
  ChartMarkerOverlay,
  type ChartMarkers,
  MarkerLayer,
  activeMarkerSpan,
  placeMarkers,
  stackGapPct,
  stackHeightPx,
} from "./chart-markers";
import { ChartError } from "./components/chart-error";
import { ChartWaveLoading } from "./components/chart-wave-loading";
import { formatBucketInterval } from "./format-interval";
import { tooltipRows } from "./tooltip-rows";

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

const Y_LABEL_GAP_PX = 8;
const Y_TICK_FONT_PX = 10;
const MUTED_STROKE = "var(--color-gray-6)";

export type ValueParts = { value: string; unit?: string; hint?: string };

export type AreaTimeseriesAxisOptions = {
  visible?: boolean;
  x?: {
    domain?: [number, number];
    contractOnSparseData?: boolean;
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
  tooltipFooter?: (point: AreaChartPoint) => ReactNode;
  focusKey?: string | null;
  connectNulls?: boolean;
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
  tooltipFooter,
  focusKey = null,
  connectNulls = false,
}: Props) {
  const handleActive = onActiveChange
    ? (state: unknown) => onActiveChange(activePointFromState(state, data))
    : undefined;
  const chartId = useId().replace(/:/g, "");
  const [shouldAnimate, setShouldAnimate] = useState(true);
  const scrub = useSyncedScrub(syncId);
  const { width: widthPx, measureRef } = useMeasuredWidth<HTMLDivElement>();
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
  const showAxes = axis !== null && axis?.visible !== false;
  const yGutterPx = showAxes
    ? Math.ceil(Math.max(...yTicks.map((t) => measureLabelPx(formatYTick(t))))) + Y_LABEL_GAP_PX
    : 0;

  let firstNonZeroTs: number | undefined;
  let lastNonZeroTs: number | undefined;
  for (const p of data) {
    if (configKeys.some((k) => Number(p[k]) > 0)) {
      firstNonZeroTs ??= p.originalTimestamp;
      lastNonZeroTs = p.originalTimestamp;
    }
  }
  const { effectiveDomain } = resolveXAxisDomain({
    xAxisDomain: axis?.x?.domain,
    contractOnSparseData: axis?.x?.contractOnSparseData,
    firstNonZeroTs,
    lastNonZeroTs,
  });

  const xDomain: [number, number] = effectiveDomain ?? [
    data[0].originalTimestamp,
    data[data.length - 1].originalTimestamp,
  ];
  const edges = axisEdges({
    startMs: xDomain[0],
    endMs: xDomain[1],
    bucketMs: data.length > 1 ? data[1].originalTimestamp - data[0].originalTimestamp : 60_000,
    nowMs: Date.now(),
  });

  const placed = markers
    ? placeMarkers(markers.items, xDomain, stackGapPct((widthPx ?? 0) - yGutterPx))
    : [];
  const margin = {
    top: (showAxes || markers ? 16 : 4) + stackHeightPx(placed),
    right: 0,
    bottom: 0,
    left: 0,
  };
  const markerSpan = markers ? activeMarkerSpan(markers.items, markers.activeId, xDomain) : null;
  const insetLeft = margin.left + yGutterPx;

  const style = FILL_STYLES[fill];
  const focused = focusKey !== null && configKeys.includes(focusKey) ? focusKey : null;
  const muted = (key: string) => focused !== null && key !== focused;
  const drawOrder = focused === null ? configKeys : [...configKeys.filter(muted), focused];
  const { showDetail } = scrub;

  const chart = (
    <ChartContainer
      config={config}
      className={cn(
        "!flex-col aspect-auto w-full [&_.recharts-surface]:overflow-visible",
        style.containerClass,
        chartContainerClassname,
      )}
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
          allowDataOverflow={
            effectiveDomain !== undefined &&
            data.some(
              (p) =>
                p.originalTimestamp < effectiveDomain[0] ||
                p.originalTimestamp > effectiveDomain[1],
            )
          }
          scale="time"
          tick={
            showAxes
              ? (props: XAxisTickContentProps) => <EdgeTick {...props} edges={edges} />
              : false
          }
          tickLine={false}
          axisLine={false}
          ticks={xDomain}
          interval={0}
          hide={!showAxes}
        />
        <YAxis
          width={yGutterPx}
          tickLine={false}
          axisLine={false}
          tick={
            showAxes
              ? ({ y, payload }: YAxisTickContentProps) => (
                  <text
                    x={0}
                    y={y}
                    dy="0.355em"
                    textAnchor="start"
                    fill="var(--color-gray-10)"
                    fontSize={Y_TICK_FONT_PX}
                  >
                    {formatYTick(Number(payload.value))}
                  </text>
                )
              : false
          }
          ticks={yTicks}
          domain={yDomain}
          hide={!showAxes}
        />
        {markers && <MarkerLayer markers={markers} span={markerSpan} placed={placed} />}
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
            const startTs = point.originalTimestamp;
            const startIndex = data.findIndex((p) => p.originalTimestamp === startTs);
            const nextTs = startIndex >= 0 ? data[startIndex + 1]?.originalTimestamp : undefined;
            const { rows, hiddenCount } = tooltipRows(point, configKeys);
            return (
              <div
                role="tooltip"
                className="grid w-max max-w-[300px] animate-in items-start gap-1.5 rounded-xl bg-raised/80 px-3 py-2.5 text-xs shadow-floating backdrop-blur-md duration-150 fade-in-0 zoom-in-95 select-none"
              >
                <div className="font-medium text-2xs text-gray-11">
                  {formatBucketInterval(startTs, nextTs, showDateInTooltip)}
                </div>
                <div className="grid gap-1">
                  {rows.map(({ key, value }) => {
                    const itemConfig = config[key];
                    const parts = formatTooltipValue(value);
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
                  {rows.length === 0 && <span className="text-gray-9">No data</span>}
                  {hiddenCount > 0 && <span className="text-gray-9">+{hiddenCount} more</span>}
                </div>
                {tooltipFooter?.(point)}
              </div>
            );
          }}
        />
        {drawOrder.map((key) => (
          <Area
            key={key}
            dataKey={key}
            type="monotone"
            connectNulls={connectNulls}
            stroke={muted(key) ? MUTED_STROKE : config[key].color}
            strokeWidth={key === focused ? 2 : 1.5}
            fill={style.gradient ? `url(#${chartId}-${key})` : config[key].color}
            fillOpacity={muted(key) ? 0 : style.fillOpacity}
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
    <div ref={measureRef} className={cn("relative w-full", widthPx === undefined && "invisible")}>
      {chart}
      <ChartMarkerOverlay
        markers={markers}
        placed={placed}
        insetLeft={insetLeft}
        insetRight={margin.right}
        bandHeight={margin.top}
      />
    </div>
  );
}

function EdgeTick({
  x,
  y,
  index,
  edges,
}: Pick<XAxisTickContentProps, "x" | "y" | "index"> & { edges: [AxisEdge, AxisEdge] }) {
  const isStart = index === 0;
  return (
    <text
      x={x}
      y={y}
      dy={10}
      textAnchor={isStart ? "start" : "end"}
      fill="var(--color-gray-10)"
      fontSize={10}
    >
      <AxisEdgeLabel edge={isStart ? edges[0] : edges[1]} />
    </text>
  );
}

let labelCanvas: CanvasRenderingContext2D | null | undefined;

function measureLabelPx(text: string): number {
  if (typeof document === "undefined") {
    return text.length * Y_TICK_FONT_PX * 0.62;
  }
  if (labelCanvas === undefined) {
    labelCanvas = document.createElement("canvas").getContext("2d");
    if (labelCanvas) {
      labelCanvas.font = `${Y_TICK_FONT_PX}px ${getComputedStyle(document.body).fontFamily}`;
    }
  }
  return labelCanvas ? labelCanvas.measureText(text).width : text.length * Y_TICK_FONT_PX * 0.62;
}
