"use client";

import { ChartContainer, ChartTooltip } from "@/components/ui/chart";
import { formatPrice } from "@/lib/fmt";
import { Skeleton } from "@unkey/ui";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { ChartError } from "./components/chart-error";

export type SpendBarSeries = { key: string; label: string; color: string };
export type SpendBarPoint = { time: number } & Record<string, number>;

export const SPEND_BAR_CHART_HEIGHT = 152;
const CATEGORY_GAP = "44%";
const PARTIAL_OPACITY = 0.35;
const TICK_COUNT = 4;
const TICK_MARGIN = 5;
// The tick labels sit flush with the right edge of the card header's 28px icon
// badge. Recharts insets a tick label by tickMargin plus 6px of its own.
const AXIS_WIDTH = 28 + TICK_MARGIN + 6;
// Stacked segments are separated by trimming their height. A stroke would eat
// into the bar's sides and change its width.
const SEGMENT_GAP = 2;

type Props = {
  data: SpendBarPoint[];
  series: SpendBarSeries[];
  incompleteFrom: number;
  isLoading?: boolean;
  isError?: boolean;
};

// Steps fit the peak instead of rounding to tens: a round domain leaves the
// tallest bar at half height and the chart reads flat. Under $4 the steps drop
// to cents so a near-idle workspace still has bars.
export function spendTicks(peakCents: number): number[] {
  if (peakCents <= 0) {
    return [0, 100];
  }
  const step =
    peakCents < 400
      ? Math.max(1, Math.ceil(peakCents / TICK_COUNT))
      : Math.ceil(peakCents / 100 / TICK_COUNT) * 100;
  return Array.from({ length: TICK_COUNT + 1 }, (_, index) => index * step);
}

export function formatAxisCents(cents: number): string {
  if (cents === 0) {
    return "$0";
  }
  return cents % 100 === 0 ? `$${(cents / 100).toLocaleString("en-US")}` : formatPrice(cents);
}

function formatDay(time: number): string {
  return new Date(time).toLocaleDateString("en-US", {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
  });
}

function isTimePoint(value: unknown): value is { time: number } {
  return (
    typeof value === "object" && value !== null && "time" in value && typeof value.time === "number"
  );
}

type SegmentProps = {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  payload?: unknown;
};

function Segment({
  x = 0,
  y = 0,
  width = 0,
  height = 0,
  color,
  gap,
  partial,
}: Omit<SegmentProps, "payload"> & { color: string; gap: number; partial: boolean }) {
  if (width <= 0 || height <= 0) {
    return <g />;
  }
  const drawn = height > gap + 1 ? height - gap : height;
  return (
    <rect
      x={x}
      y={y}
      width={width}
      height={drawn}
      fill={color}
      fillOpacity={partial ? PARTIAL_OPACITY : 1}
    />
  );
}

export function SpendBarChart({ data, series, incompleteFrom, isLoading, isError }: Props) {
  if (isError) {
    return <ChartError height={SPEND_BAR_CHART_HEIGHT} />;
  }
  if (isLoading) {
    return <Skeleton className="w-full rounded-md" style={{ height: SPEND_BAR_CHART_HEIGHT }} />;
  }

  const pointAt = (payload: unknown) =>
    isTimePoint(payload) ? data.find((point) => point.time === payload.time) : undefined;
  const peak = data.reduce(
    (max, point) =>
      Math.max(
        max,
        series.reduce((sum, entry) => sum + point[entry.key], 0),
      ),
    0,
  );
  const ticks = spendTicks(peak);
  const top = ticks[ticks.length - 1];

  return (
    <ChartContainer
      config={{}}
      role="img"
      aria-label="Daily compute spend by project"
      className="!flex-col aspect-auto w-full"
      style={{ height: SPEND_BAR_CHART_HEIGHT, width: "100%" }}
    >
      <BarChart
        data={data}
        barCategoryGap={CATEGORY_GAP}
        margin={{ top: 8, right: 0, bottom: 0, left: 0 }}
      >
        {/* CartesianGrid derives its own ticks (four against the axis's five) and
            recharts 3.7 ignores horizontalValues and syncWithTicks, so the lines
            are placed from the same ticks the axis labels use. */}
        <CartesianGrid
          vertical={false}
          horizontalCoordinatesGenerator={({ offset }) =>
            ticks.map((tick) => offset.top + offset.height * (1 - tick / top))
          }
          stroke="hsl(var(--gray-4))"
          strokeWidth={1}
        />
        <XAxis
          dataKey="time"
          type="category"
          tickFormatter={formatDay}
          tick={{ fill: "hsl(var(--gray-10))", fontSize: 10 }}
          tickLine={false}
          axisLine={false}
          interval="preserveStartEnd"
          minTickGap={24}
          height={20}
        />
        <YAxis
          width={AXIS_WIDTH}
          tickMargin={TICK_MARGIN}
          domain={[0, top]}
          ticks={ticks}
          tickFormatter={formatAxisCents}
          tick={{ fill: "hsl(var(--gray-10))", fontSize: 10 }}
          tickLine={false}
          axisLine={false}
        />
        <ChartTooltip
          allowEscapeViewBox={{ x: false, y: true }}
          wrapperStyle={{ zIndex: 1000, pointerEvents: "none" }}
          cursor={{ fill: "hsl(var(--grayA-3))" }}
          content={({ active, payload }) => {
            const point = pointAt(payload?.[0]?.payload);
            if (!active || point === undefined) {
              return null;
            }
            return (
              <SpendTooltip point={point} series={series} partial={point.time >= incompleteFrom} />
            );
          }}
        />
        {series.map((entry, index) => (
          <Bar
            key={entry.key}
            dataKey={entry.key}
            stackId="spend"
            isAnimationActive={false}
            activeBar={false}
            shape={({ payload, ...rect }: SegmentProps) => {
              const point = pointAt(payload);
              const stacked =
                point !== undefined && series.slice(0, index).some((below) => point[below.key] > 0);
              return (
                <Segment
                  {...rect}
                  color={entry.color}
                  gap={stacked ? SEGMENT_GAP : 0}
                  partial={point !== undefined && point.time >= incompleteFrom}
                />
              );
            }}
          />
        ))}
      </BarChart>
    </ChartContainer>
  );
}

function SpendTooltip({
  point,
  series,
  partial,
}: {
  point: SpendBarPoint;
  series: SpendBarSeries[];
  partial: boolean;
}) {
  const contributors = series
    .map((entry) => ({ ...entry, cents: point[entry.key] }))
    .filter((entry) => entry.cents > 0)
    .sort((a, b) => b.cents - a.cents);
  if (contributors.length === 0) {
    return null;
  }
  const total = contributors.reduce((sum, entry) => sum + entry.cents, 0);

  return (
    <div
      role="tooltip"
      className="grid w-max min-w-[200px] animate-in gap-1.5 rounded-xl border border-gray-4/50 bg-gray-1/80 px-3 py-2.5 text-xs shadow-2xl backdrop-blur-md duration-150 fade-in-0 zoom-in-95 select-none"
    >
      <div className="font-medium text-[11px] text-accent-11">
        {formatDay(point.time)}
        {partial ? <span className="pl-1 font-normal text-gray-10">so far</span> : null}
      </div>
      <div className="grid gap-1">
        {contributors.map((entry) => (
          <div key={entry.key} className="flex items-center gap-2">
            <span
              className="size-2 shrink-0 rounded-full"
              style={{ backgroundColor: entry.color }}
              aria-hidden="true"
            />
            <span className="truncate text-accent-12">{entry.label}</span>
            <span className="ml-auto font-mono text-accent-12 tabular-nums">
              {formatPrice(entry.cents)}
            </span>
          </div>
        ))}
      </div>
      {contributors.length > 1 ? (
        <div className="flex items-center gap-2 border-grayA-4 border-t pt-1.5">
          <span className="text-gray-11">Total</span>
          <span className="ml-auto font-medium font-mono text-accent-12 tabular-nums">
            {formatPrice(total)}
          </span>
        </div>
      ) : null}
    </div>
  );
}
