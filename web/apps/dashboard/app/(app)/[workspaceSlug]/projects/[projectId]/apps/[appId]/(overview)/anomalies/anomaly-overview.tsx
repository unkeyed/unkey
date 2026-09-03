"use client";

import { cn } from "@/lib/utils";
import { Skeleton } from "@unkey/ui";
import { dayMs } from "./range";

type Point = { time: number; value: number };
const skeletonDays = ["one", "two", "three", "four", "five", "six", "seven"] as const;

export function AnomalyOverview({
  buckets,
  selectedRange,
  loading,
  onSelectRange,
}: {
  buckets: Point[];
  selectedRange: { startMs: number; endMs: number };
  loading: boolean;
  onSelectRange: (range: { startMs: number; endMs: number }) => void;
}) {
  if (loading) {
    return (
      <div className="grid grid-cols-7 gap-1.5" aria-label="Loading seven-day overview">
        {skeletonDays.map((day) => (
          <Skeleton key={day} className="h-[78px] rounded-md" />
        ))}
      </div>
    );
  }

  const lastBucket = buckets.at(-1)?.time ?? selectedRange.endMs;
  const finalDayStart = startOfDay(lastBucket);
  const days = Array.from({ length: 7 }, (_, index) => {
    const startMs = finalDayStart - (6 - index) * dayMs;
    return {
      startMs,
      endMs: startMs + dayMs,
      points: buckets.filter((point) => point.time >= startMs && point.time < startMs + dayMs),
    };
  });

  return (
    <div
      className="grid grid-cols-2 gap-1.5 sm:grid-cols-4 lg:grid-cols-7"
      aria-label="Seven-day overview"
    >
      {days.map((day) => {
        const selected = day.startMs < selectedRange.endMs && day.endMs > selectedRange.startMs;
        return (
          <button
            key={day.startMs}
            type="button"
            onClick={() => onSelectRange({ startMs: day.startMs, endMs: day.endMs })}
            className={cn(
              "group flex h-[78px] flex-col overflow-hidden rounded-md border bg-gray-1 text-left transition-colors",
              selected
                ? "border-infoA-7 bg-infoA-2"
                : "border-grayA-4 hover:border-grayA-7 hover:bg-grayA-2",
            )}
          >
            <span className="px-2 pt-1.5 text-[11px] font-medium text-gray-10">
              {formatDay(day.startMs)}
            </span>
            <svg
              viewBox="0 0 100 38"
              preserveAspectRatio="none"
              className="mt-auto h-11 w-full"
              aria-hidden="true"
            >
              <polyline
                points={sparklinePoints(day.points)}
                fill="none"
                stroke="hsl(var(--info-9))"
                strokeWidth="1.5"
                vectorEffect="non-scaling-stroke"
              />
            </svg>
          </button>
        );
      })}
    </div>
  );
}

function startOfDay(value: number): number {
  const date = new Date(value);
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

function formatDay(value: number): string {
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }).format(value);
}

function sparklinePoints(points: Point[]): string {
  if (points.length === 0) {
    return "0,34 100,34";
  }
  const maxValue = Math.max(...points.map((point) => point.value), 1);
  const denominator = Math.max(points.length - 1, 1);
  return points
    .map((point, index) => {
      const x = (index / denominator) * 100;
      const y = 34 - (point.value / maxValue) * 28;
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    })
    .join(" ");
}
