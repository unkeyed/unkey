"use client";

import { cn } from "@/lib/utils";
import { type MouseEvent, useState } from "react";

export type Mark = "line" | "bars" | "ratio" | "heatmap";

/** Names for the two series in a hover readout, e.g. Valid / Invalid. */
export type SeriesLabels = { ok: string; bad: string };

const DEFAULT_LABELS: SeriesLabels = { ok: "ok", bad: "errors" };

function fmtCount(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(2).replace(/\.?0+$/, "")}M`;
  }
  if (n >= 1_000) {
    return `${Math.round(n / 1_000)}K`;
  }
  return String(n);
}

// Buckets are hourly and the last one is the current hour, so the index is
// enough to label a point without threading timestamps through every caller.
function hoursAgoLabel(index: number, total: number): string {
  const hoursAgo = total - 1 - index;
  if (hoursAgo === 0) {
    return "This hour";
  }
  return `${hoursAgo}h ago`;
}

// Default error colour when a caller doesn't name one. Rows that know their data
// type pass their own so ratelimits don't paint with the verification family.
const DEFAULT_ERROR = "hsl(var(--chart-verify-bad))";

function lastN(points: number[], n: number): number[] {
  return points.length > n ? points.slice(points.length - n) : points;
}

function lastN30Buckets(buckets: Bucket[]): Bucket[] {
  return buckets.length > 30 ? buckets.slice(buckets.length - 30) : buckets;
}

export type Bucket = { valid: number; error: number };

export function RowMark({
  mark,
  points,
  buckets,
  errorRatio,
  stroke,
  errorStroke,
  labels,
  className,
  fill,
}: {
  mark: Mark;
  points: number[];
  /** Per-bucket valid/error split. Preferred over the flat errorRatio: a single
   *  ratio applied to every bar draws an even orange rim, which no real error
   *  data ever does. */
  buckets?: Bucket[];
  errorRatio: number;
  stroke: string;
  /** Colour for the error share; defaults to the verification family. */
  errorStroke?: string;
  /** Series names for the hover readout. */
  labels?: SeriesLabels;
  className?: string;
  /** Bars only: spread across the given width rather than packing them. */
  fill?: boolean;
}) {
  if (points.length === 0 && mark !== "ratio") {
    return <div className={cn("rounded bg-grayA-3", className)} />;
  }
  switch (mark) {
    case "bars":
      return (
        <BarsMark
          points={points}
          buckets={buckets}
          errorRatio={errorRatio}
          stroke={stroke}
          errorStroke={errorStroke}
          labels={labels}
          className={className}
          fill={fill}
        />
      );
    case "ratio":
      return (
        <RatioMark
          errorRatio={errorRatio}
          stroke={stroke}
          errorStroke={errorStroke}
          className={className}
        />
      );
    case "heatmap":
      return <HeatmapMark points={points} stroke={stroke} className={className} />;
    default:
      return <LineMark points={points} stroke={stroke} className={className} />;
  }
}

// Full-width background chart for the hybrid row: a smooth line with a gradient
// area that fades upward, so text above it stays readable.
export function HybridChart({
  points,
  stroke,
  className,
}: {
  points: number[];
  stroke: string;
  className?: string;
}) {
  if (points.length === 0) {
    return null;
  }
  const w = 800;
  const h = 100;
  const pad = 10;
  const max = Math.max(...points);
  const min = Math.min(...points);
  const range = max - min || 1;
  const step = points.length > 1 ? w / (points.length - 1) : w;
  const coords = points.map((p, i) => {
    const x = i * step;
    const y = h - pad - ((p - min) / range) * (h - pad * 2);
    return [x, y] as const;
  });
  let d = `M${coords[0][0]},${coords[0][1]}`;
  for (let i = 1; i < coords.length; i++) {
    const [px, py] = coords[i - 1];
    const [x, y] = coords[i];
    d += ` Q${px},${py} ${(px + x) / 2},${(py + y) / 2}`;
  }
  d += ` L${coords[coords.length - 1][0]},${coords[coords.length - 1][1]}`;
  const area = `${d} L${w},${h} L0,${h} Z`;
  const gid = `hybgrad-${stroke.replace(/[^a-z0-9]/gi, "")}`;
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className={className} preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={stroke} stopOpacity="0" />
          <stop offset="100%" stopColor={stroke} stopOpacity="0.22" />
        </linearGradient>
      </defs>
      <path d={area} fill={`url(#${gid})`} />
      <path
        d={d}
        fill="none"
        stroke={stroke}
        strokeOpacity="0.5"
        strokeWidth="1.5"
        vectorEffect="non-scaling-stroke"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function LineMark({
  points,
  stroke,
  className,
}: {
  points: number[];
  stroke: string;
  className?: string;
}) {
  const w = 120;
  const h = 40;
  const padY = 7;
  const max = Math.max(...points);
  const min = Math.min(...points);
  const range = max - min || 1;
  const step = points.length > 1 ? w / (points.length - 1) : w;
  const coords = points.map((p, i) => {
    const x = i * step;
    const y = h - padY - ((p - min) / range) * (h - padY * 2);
    return [x, y] as const;
  });
  let d = `M${coords[0][0]},${coords[0][1]}`;
  for (let i = 1; i < coords.length; i++) {
    const [px, py] = coords[i - 1];
    const [x, y] = coords[i];
    const mx = (px + x) / 2;
    const my = (py + y) / 2;
    d += ` Q${px},${py} ${mx},${my}`;
  }
  const [lastX, lastY] = coords[coords.length - 1];
  d += ` L${lastX},${lastY}`;
  const area = `${d} L${w},${h} L0,${h} Z`;
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className={className} preserveAspectRatio="none" aria-hidden>
      <path d={area} fill={stroke} fillOpacity={0.12} />
      <path
        d={d}
        fill="none"
        stroke={stroke}
        strokeWidth={2}
        vectorEffect="non-scaling-stroke"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx={lastX} cy={lastY} r={2} fill={stroke} vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

// Mirrors the real listing-page sparkline (components/stats-list-card): bars
// sit directly on the row background, no well, so the accent-4 fill stays visible.
function BarsMark({
  points,
  buckets,
  errorRatio,
  stroke,
  errorStroke = DEFAULT_ERROR,
  labels,
  className,
  fill,
}: {
  points: number[];
  buckets?: Bucket[];
  errorRatio: number;
  stroke: string;
  errorStroke?: string;
  labels?: SeriesLabels;
  className?: string;
  /** Spread buckets across the given width instead of packing them at 3px+2px. */
  fill?: boolean;
}) {
  const split = buckets ? lastN30Buckets(buckets) : undefined;
  const data = split ? split.map((b) => b.valid + b.error) : lastN(points, 30);
  const H = 28;
  const max = Math.max(...data, 1) * 1.3;
  // Hover readouts need real per-bucket numbers, so only charts fed by buckets
  // get them — a bare `spark` array carries magnitudes with no valid/error split.
  const [hovered, setHovered] = useState<number | null>(null);
  const series = labels ?? DEFAULT_LABELS;
  const active = split && hovered !== null ? split[hovered] : undefined;
  // Anchor the readout over its bucket, then slide it back by the same fraction
  // so the near edge never leaves the chart — no clipping at either end.
  const anchorPct = hovered === null ? 0 : ((hovered + 0.5) / data.length) * 100;
  // Tracked on the container rather than per-band: the bands are ~6px wide, and
  // enter/leave on targets that small drops events as the pointer crosses them.
  const track = (event: MouseEvent<HTMLDivElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    const index = Math.floor(((event.clientX - rect.left) / rect.width) * data.length);
    setHovered(Math.min(data.length - 1, Math.max(0, index)));
  };
  return (
    <div
      className={cn(
        "relative items-end",
        // Default is the original packed layout: fixed 3px bars, 2px apart,
        // shrink-wrapped. `fill` opts into equal bands that span the given width.
        fill ? "flex h-7 gap-0" : "inline-flex h-7 gap-[2px]",
        fill && className,
      )}
      onMouseMove={split ? track : undefined}
      onMouseLeave={split ? () => setHovered(null) : undefined}
    >
      {active && hovered !== null && (
        <div
          className="pointer-events-none absolute bottom-full z-20 mb-1.5 whitespace-nowrap rounded-md border border-grayA-4 bg-gray-1 px-2 py-1.5 shadow-lg dark:bg-black"
          style={{ left: `${anchorPct}%`, transform: `translateX(-${anchorPct}%)` }}
        >
          <div className="text-[11px] text-gray-9">{hoursAgoLabel(hovered, data.length)}</div>
          <div className="mt-1 flex items-center gap-3">
            <span className="flex items-center gap-1.5">
              <span
                className="h-2.5 w-1 shrink-0 rounded"
                style={{ backgroundColor: stroke }}
                aria-hidden
              />
              <span className="text-[11px] tabular-nums text-accent-12">
                {fmtCount(active.valid)}
              </span>
              <span className="text-[11px] lowercase text-gray-9">{series.ok}</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span
                className="h-2.5 w-1 shrink-0 rounded"
                style={{ backgroundColor: errorStroke }}
                aria-hidden
              />
              <span className="text-[11px] tabular-nums text-accent-12">
                {fmtCount(active.error)}
              </span>
              <span className="text-[11px] lowercase text-gray-9">{series.bad}</span>
            </span>
          </div>
        </div>
      )}
      {data.map((v, i) => {
        const total = Math.min(Math.round((v / max) * H), H);
        const ratio = split ? (v > 0 ? split[i].error / v : 0) : errorRatio;
        // Per-bucket splits skip the 1px floor: at a fraction of a percent the
        // cap rounds to sub-pixel, and forcing it to 1px paints a solid orange
        // rim across every bar — "every bucket is failing" at ~6x the real rate.
        // Flat-ratio callers keep the floor so a small error share stays visible.
        const scaled = ratio * total;
        const top = ratio > 0 && v > 0 ? (split ? Math.round(scaled) : Math.max(scaled, 1)) : 0;
        const bottom = Math.max(total - top, 0);
        return (
          <div
            // biome-ignore lint/suspicious/noArrayIndexKey: positional bars
            key={i}
            className={cn(
              // h-full so the hover wash spans the chart, not just the bar.
              "relative flex h-full flex-col items-center justify-end",
              fill ? "min-w-px flex-1" : "w-[3px] shrink-0",
            )}
          >
            {/* The band is only as wide as its bar, so the hover wash has to
                bleed into the gaps to be visible at all. It stays under the
                bars by painting first — both are positioned, so DOM order wins
                (a negative z-index would drop it behind the card background). */}
            {hovered === i && (
              <div className="pointer-events-none absolute -left-px -right-px bottom-0 top-0 bg-grayA-5" />
            )}
            <div
              className="relative w-[3px] max-w-full"
              style={{ height: `${top}px`, backgroundColor: errorStroke }}
            />
            <div
              className="relative w-[3px] max-w-full"
              style={{ height: `${bottom}px`, backgroundColor: stroke }}
            />
            {/* One tick per bucket instead of a dashed rule across the whole
                width: a continuous line breaks against every bar base, and
                because the tick lives inside the band it lines up with its bar
                by construction rather than by luck. */}
            <div className={cn("relative h-px bg-gray-5", fill ? "w-[5px]" : "w-[4px]")} />
          </div>
        );
      })}
    </div>
  );
}

function RatioMark({
  errorRatio,
  stroke,
  errorStroke = DEFAULT_ERROR,
  className,
}: {
  errorRatio: number;
  stroke: string;
  errorStroke?: string;
  className?: string;
}) {
  const validPct = Math.max(0, Math.min(100, (1 - errorRatio) * 100));
  return (
    <div className={cn("flex items-center", className)}>
      <div className="w-full h-2 rounded-full overflow-hidden flex bg-grayA-3">
        <div style={{ width: `${validPct}%`, backgroundColor: stroke }} className="h-full" />
        <div
          style={{ width: `${100 - validPct}%`, backgroundColor: errorStroke }}
          className="h-full"
        />
      </div>
    </div>
  );
}

function HeatmapMark({
  points,
  stroke,
  className,
}: {
  points: number[];
  stroke: string;
  className?: string;
}) {
  const data = lastN(points, 12);
  const max = Math.max(...data, 1);
  return (
    <div className={cn("flex items-center gap-[2px]", className)}>
      {data.map((v, i) => (
        <div
          // biome-ignore lint/suspicious/noArrayIndexKey: positional cells
          key={i}
          className="h-4 flex-1 rounded-[2px]"
          style={{ backgroundColor: stroke, opacity: 0.15 + 0.85 * (v / max) }}
        />
      ))}
    </div>
  );
}
