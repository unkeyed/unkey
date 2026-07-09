"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatNumber } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { InfoTooltip } from "@unkey/ui";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Bar, BarChart, Tooltip as RechartsTooltip, ResponsiveContainer, YAxis } from "recharts";
import {
  type VerificationTimeseriesPoint,
  useFetchVerificationTimeseries,
} from "./hooks/use-query-timeseries";

const CURSOR_WIDTH = 14;
const EMPTY_TICK_COUNT = 12;

type ChartState =
  | { type: "loading" }
  | { type: "error" }
  | { type: "empty"; buckets: number }
  | { type: "data"; points: VerificationTimeseriesPoint[] };

type Props = {
  api: ApiOverview;
};

export function ApiListCard({ api }: Props) {
  const { timeseries, isLoading, isError } = useFetchVerificationTimeseries(api.keyspaceId);
  const workspace = useWorkspaceNavigation();

  const passed = timeseries?.reduce((acc, point) => acc + point.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, point) => acc + point.error, 0) ?? 0;

  const chart: ChartState = isError
    ? { type: "error" }
    : isLoading || !timeseries
      ? { type: "loading" }
      : passed + blocked > 0
        ? { type: "data", points: timeseries }
        : { type: "empty", buckets: timeseries.length || EMPTY_TICK_COUNT };

  return (
    <Link
      href={routes.apis.detail({ workspaceSlug: workspace.slug, apiId: api.id })}
      aria-label={`View ${api.name} API`}
      className="relative h-full p-5 flex flex-col border border-grayA-4 hover:border-grayA-7 rounded-lg w-full gap-5 transition-all duration-300"
    >
      <div className="flex flex-col w-full gap-2 min-w-0">
        <InfoTooltip content={api.name} asChild position={{ align: "start", side: "top" }}>
          <span className="font-medium text-sm leading-[14px] text-accent-12 truncate">
            {api.name}
          </span>
        </InfoTooltip>
        <InfoTooltip content={api.id} asChild position={{ align: "start", side: "top" }}>
          <span className="font-mono text-xs leading-[12px] text-gray-11 truncate">{api.id}</span>
        </InfoTooltip>
      </div>

      <div className="mt-auto flex flex-col gap-3">
        <ChartWell chart={chart} />
        <div className="flex gap-3 items-center min-w-0 text-xs text-gray-11">
          <span>
            <span className="tabular-nums">{formatNumber(api.keyCount)}</span>{" "}
            {api.keyCount === 1 ? "key" : "keys"}
          </span>
          {chart.type === "data" ? (
            <div className="ml-auto flex items-center gap-3">
              <span className="flex items-center gap-1.5">
                <span className="bg-accent-8 rounded h-[10px] w-1 shrink-0" />
                <span>
                  <span className="tabular-nums">{formatNumber(passed)}</span> valid
                </span>
              </span>
              <span className="flex items-center gap-1.5">
                <span className="bg-orange-9 rounded h-[10px] w-1 shrink-0" />
                <span>
                  <span className="tabular-nums">{formatNumber(blocked)}</span> invalid
                </span>
              </span>
            </div>
          ) : null}
        </div>
      </div>
    </Link>
  );
}

function ChartWell({ chart }: { chart: ChartState }) {
  return (
    <div className="relative h-12 w-full">
      <div className="absolute inset-x-0 bottom-0 border-t border-dashed border-gray-5 pointer-events-none" />
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
        <>
          <BaselineTicks buckets={chart.points.length} />
          <ApiSparkline data={chart.points} />
        </>
      )}
    </div>
  );
}

// Grid cells mirror the chart's evenly-spaced bar bands: ticks stay visible under
// quiet buckets, and empty cards get the same bucket-width hover highlight as the
// chart's cursor without any mousemove tracking.
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

function ApiSparkline({ data }: { data: VerificationTimeseriesPoint[] }) {
  const realMax = Math.max(1, ...data.map((d) => d.success + d.error));
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
    const onPointerMove = (e: PointerEvent) => {
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) {
        return;
      }
      const inside =
        e.clientX >= rect.left &&
        e.clientX <= rect.right &&
        e.clientY >= rect.top &&
        e.clientY <= rect.bottom;
      if (!inside) {
        setActive(false);
      }
    };
    document.addEventListener("pointermove", onPointerMove, true);
    return () => document.removeEventListener("pointermove", onPointerMove, true);
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
          data={data}
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
              const point: VerificationTimeseriesPoint | undefined = payload[0]?.payload;
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
                        <span>Valid</span>
                      </span>
                      <span className="tabular-nums">{formatNumber(point.success)}</span>
                    </div>
                    <div className="flex items-center justify-between gap-4">
                      <span className="flex items-center gap-1.5">
                        <span className="bg-orange-9 w-1 h-2.5 rounded-sm" />
                        <span>Invalid</span>
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
            dataKey="success"
            stackId="a"
            fill="hsl(var(--accent-4))"
            activeBar={{ fill: "hsl(var(--accent-7))" }}
            maxBarSize={8}
            isAnimationActive={false}
          />
          <Bar
            dataKey="error"
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
