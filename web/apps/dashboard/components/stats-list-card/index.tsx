"use client";

import { formatNumber } from "@/lib/fmt";
import { InfoTooltip } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { type ReactNode, useEffect, useRef, useState } from "react";
import { Bar, BarChart, Tooltip as RechartsTooltip, ResponsiveContainer, YAxis } from "recharts";

const CURSOR_WIDTH = 14;
const EMPTY_TICK_COUNT = 12;
// Below this share of the busiest bucket a bar is only ~2-5px on the 48px well and
// reads as noise, so we hide it.
const MIN_VISIBLE_BAR_RATIO = 0.15;

export type StatsListCardBucket = {
  displayX: string;
  success: number;
  error: number;
};

export type StatsListCardLabels = {
  success: string;
  error: string;
};

type ChartState =
  | { type: "loading" }
  | { type: "error" }
  | { type: "empty"; buckets: number }
  | { type: "data"; points: StatsListCardBucket[] };

export type StatsListCardProps = {
  href: Route;
  ariaLabel: string;
  title: string;
  subtitle?: string;
  buckets: StatsListCardBucket[] | undefined;
  isLoading: boolean;
  isError: boolean;
  labels: StatsListCardLabels;
  footerLeft: ReactNode;
};

export function StatsListCard({
  href,
  ariaLabel,
  title,
  subtitle,
  buckets,
  isLoading,
  isError,
  labels,
  footerLeft,
}: StatsListCardProps) {
  const success = buckets?.reduce((acc, point) => acc + point.success, 0) ?? 0;
  const error = buckets?.reduce((acc, point) => acc + point.error, 0) ?? 0;

  const chart: ChartState = isError
    ? { type: "error" }
    : isLoading || !buckets
      ? { type: "loading" }
      : success + error > 0
        ? { type: "data", points: buckets }
        : { type: "empty", buckets: buckets.length || EMPTY_TICK_COUNT };

  return (
    <Link
      href={href}
      aria-label={ariaLabel}
      className="relative h-full p-5 flex flex-col border border-grayA-4 hover:border-grayA-7 rounded-lg w-full gap-5 transition-all duration-300"
    >
      <div className="flex flex-col w-full gap-2 min-w-0">
        <InfoTooltip content={title} asChild position={{ align: "start", side: "top" }}>
          <span className="font-medium text-sm leading-[14px] text-accent-12 truncate">
            {title}
          </span>
        </InfoTooltip>
        {subtitle ? (
          <InfoTooltip content={subtitle} asChild position={{ align: "start", side: "top" }}>
            <span className="font-mono text-xs leading-[12px] text-gray-11 truncate">
              {subtitle}
            </span>
          </InfoTooltip>
        ) : null}
      </div>

      <div className="mt-auto flex flex-col gap-3">
        <ChartWell chart={chart} labels={labels} />
        <div className="flex gap-3 items-center min-w-0 text-xs text-gray-11">
          {footerLeft}
          {chart.type === "data" ? (
            <div className="ml-auto flex items-center gap-3">
              <span className="flex items-center gap-1.5">
                <span className="bg-accent-4 rounded h-[10px] w-1 shrink-0" />
                <span>
                  <span className="tabular-nums">{formatNumber(success)}</span>{" "}
                  <span className="lowercase">{labels.success}</span>
                </span>
              </span>
              <span className="flex items-center gap-1.5">
                <span className="bg-orange-9 rounded h-[10px] w-1 shrink-0" />
                <span>
                  <span className="tabular-nums">{formatNumber(error)}</span>{" "}
                  <span className="lowercase">{labels.error}</span>
                </span>
              </span>
            </div>
          ) : null}
        </div>
      </div>
    </Link>
  );
}

function ChartWell({ chart, labels }: { chart: ChartState; labels: StatsListCardLabels }) {
  return (
    <div className="relative h-12 w-full">
      {chart.type === "loading" ? (
        <div className="absolute inset-0 flex items-center justify-center text-[11px] text-gray-9 pointer-events-none">
          Loading...
        </div>
      ) : chart.type === "error" ? (
        <div className="absolute inset-0 flex items-center justify-center text-[11px] text-gray-9 pointer-events-none">
          Activity unavailable
        </div>
      ) : chart.type === "empty" ? (
        <>
          <BaselineTicks buckets={chart.buckets} />
          <div className="absolute inset-0 flex items-center justify-center text-[11px] text-gray-9 pointer-events-none">
            No activity
          </div>
        </>
      ) : (
        <StatsSparkline data={chart.points} labels={labels} />
      )}
      {/* Painted last so the dashed baseline reads in front of the bar bases. */}
      <div className="absolute inset-x-0 bottom-0 border-t border-dashed border-gray-5 pointer-events-none" />
    </div>
  );
}

// Empty-state placeholder only. Not drawn under real bars, where this CSS grid
// can't stay aligned with Recharts' own band scale.
function BaselineTicks({ buckets }: { buckets: number }) {
  return (
    <div
      className="absolute inset-0 grid"
      style={{ gridTemplateColumns: `repeat(${buckets}, minmax(0, 1fr))` }}
    >
      {Array.from({ length: buckets }).map((_, i) => (
        <div
          // biome-ignore lint/suspicious/noArrayIndexKey: ticks are purely positional
          key={i}
          className="flex items-end justify-center rounded-sm hover:bg-accent-3"
        >
          <div className="h-0.5 w-2 max-w-full bg-gray-5" />
        </div>
      ))}
    </div>
  );
}

function NarrowCursor(props: { x?: number; y?: number; width?: number; height?: number }) {
  const { x = 0, y = 0, width = 0, height = 0 } = props;
  const cx = x + width / 2;
  return (
    <rect
      x={cx - CURSOR_WIDTH / 2}
      y={y}
      width={CURSOR_WIDTH}
      height={height}
      fill="hsl(var(--accent-3))"
      opacity={0.6}
      rx={2}
    />
  );
}

function StatsSparkline({
  data,
  labels,
}: {
  data: StatsListCardBucket[];
  labels: StatsListCardLabels;
}) {
  const realMax = Math.max(1, ...data.map((d) => d.success + d.error));
  const minVisible = realMax * MIN_VISIBLE_BAR_RATIO;
  // Keep success/error on the datum so the tooltip still reports real counts for
  // buckets whose bar is hidden.
  const chartData = data.map((d) => {
    const visible = d.success + d.error >= minVisible;
    return { ...d, barSuccess: visible ? d.success : 0, barError: visible ? d.error : 0 };
  });
  // On a fast drag/swipe out of the chart, Recharts (v3) never delivers a leave
  // event — pointer capture during the drag swallows the boundary crossing — so its
  // tooltip stays stuck open. Drive visibility off our own state and, while open,
  // watch pointer position document-wide: any move whose coordinates fall outside the
  // chart closes it, which catches the swipe even when no leave event fires.
  const [active, setActive] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const close = () => setActive(false);

  useEffect(() => {
    if (!active) {
      return;
    }
    // Read the chart bounds once when the tooltip opens rather than on every move,
    // so the document-wide listener doesn't force a layout on each pointer event.
    const rect = containerRef.current?.getBoundingClientRect();
    if (!rect) {
      return;
    }
    const onPointerMove = (e: PointerEvent) => {
      const inside =
        e.clientX >= rect.left &&
        e.clientX <= rect.right &&
        e.clientY >= rect.top &&
        e.clientY <= rect.bottom;
      if (!inside) {
        setActive(false);
      }
    };
    const options = { capture: true, passive: true };
    document.addEventListener("pointermove", onPointerMove, options);
    return () => document.removeEventListener("pointermove", onPointerMove, options);
  }, [active]);

  return (
    <div
      ref={containerRef}
      className="absolute inset-0"
      onMouseLeave={close}
      onPointerLeave={close}
    >
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={chartData}
          margin={{ top: 4, right: 0, left: 0, bottom: 0 }}
          barCategoryGap="22%"
          onMouseMove={(state) => setActive(Boolean(state?.isTooltipActive))}
          onMouseLeave={close}
        >
          <YAxis hide domain={[0, realMax * 1.3]} />
          <RechartsTooltip
            active={active}
            cursor={<NarrowCursor />}
            wrapperStyle={{ outline: "none", zIndex: 20 }}
            allowEscapeViewBox={{ x: true, y: true }}
            content={({ active, payload }) => {
              if (!active || !payload?.length) {
                return null;
              }
              const point: StatsListCardBucket | undefined = payload[0]?.payload;
              if (!point) {
                return null;
              }
              return (
                <div className="px-2.5 py-2 bg-gray-12 text-gray-1 text-[11px] rounded shadow-lg whitespace-nowrap">
                  <div className="font-medium opacity-80 mb-1.5">{point.displayX}</div>
                  <div className="flex flex-col gap-1">
                    <div className="flex items-center justify-between gap-4">
                      <span className="flex items-center gap-1.5">
                        <span className="bg-accent-4 w-1 h-2.5 rounded-sm" />
                        <span>{labels.success}</span>
                      </span>
                      <span className="tabular-nums">{formatNumber(point.success)}</span>
                    </div>
                    <div className="flex items-center justify-between gap-4">
                      <span className="flex items-center gap-1.5">
                        <span className="bg-orange-9 w-1 h-2.5 rounded-sm" />
                        <span>{labels.error}</span>
                      </span>
                      <span className="tabular-nums">{formatNumber(point.error)}</span>
                    </div>
                    <div className="flex items-center justify-between gap-4 mt-1 pt-1.5 border-t border-white/15">
                      <span className="opacity-70">Total</span>
                      <span className="tabular-nums">
                        {formatNumber(point.success + point.error)}
                      </span>
                    </div>
                  </div>
                </div>
              );
            }}
          />
          <Bar
            dataKey="barSuccess"
            stackId="a"
            fill="hsl(var(--accent-4))"
            activeBar={{ fill: "hsl(var(--accent-7))" }}
            maxBarSize={8}
            isAnimationActive={false}
          />
          <Bar
            dataKey="barError"
            stackId="a"
            fill="hsl(var(--orange-9))"
            activeBar={{ fill: "hsl(var(--orange-10))" }}
            maxBarSize={8}
            isAnimationActive={false}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
