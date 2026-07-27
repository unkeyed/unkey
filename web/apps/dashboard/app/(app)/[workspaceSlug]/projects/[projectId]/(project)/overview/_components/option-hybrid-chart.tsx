"use client";

import { useId } from "react";

const REQUEST_STROKE = "hsl(var(--activity))";
const ERROR_STROKE = "hsl(var(--error-9))";

type SmoothPath = { area: string; line: string };

// Same quadratic-midpoint smoothing both path builders below share, given a
// column of already-scaled y coordinates.
function pathFromY(ys: number[], w: number, h: number): SmoothPath {
  const step = ys.length > 1 ? w / (ys.length - 1) : w;
  const coords = ys.map((y, i) => [i * step, y] as const);
  let line = `M${coords[0][0]},${coords[0][1]}`;
  for (let i = 1; i < coords.length; i++) {
    const [px, py] = coords[i - 1];
    const [x, y] = coords[i];
    line += ` Q${px},${py} ${(px + x) / 2},${(py + y) / 2}`;
  }
  const [lastX, lastY] = coords[coords.length - 1];
  line += ` L${lastX},${lastY}`;
  const area = `${line} L${w},${h} L0,${h} Z`;
  return { area, line };
}

// Normalized to its own local min/max so a single row's curve always fills
// its own box regardless of absolute magnitude. Only ever used for the
// "primary" series (valid) — an error series must NOT use this, since its
// own tiny local range would stretch it to the same visual amplitude as
// valid regardless of how small it actually is (see buildScaledPath).
function buildSmoothPath(values: number[], w: number, h: number, padY: number): SmoothPath {
  if (values.length === 0) {
    return { area: `M0,${h} L${w},${h} Z`, line: `M0,${h} L${w},${h}` };
  }
  const max = Math.max(...values);
  const min = Math.min(...values);
  const range = max - min || 1;
  const ys = values.map((v) => h - padY - ((v - min) / range) * (h - padY * 2));
  return pathFromY(ys, w, h);
}

// Renders a series against an externally supplied ceiling (the valid
// series' own peak) instead of its own local min/max, floored at 0 — so an
// error line's amplitude reflects its real share of the valid line's
// height rather than being independently stretched to fill the box.
function buildScaledPath(
  values: number[],
  w: number,
  h: number,
  padY: number,
  ceiling: number,
): SmoothPath {
  if (values.length === 0) {
    return { area: `M0,${h} L${w},${h} Z`, line: `M0,${h} L${w},${h}` };
  }
  const max = ceiling || 1;
  const ys = values.map((v) => h - padY - (v / max) * (h - padY * 2));
  return pathFromY(ys, w, h);
}

// Below this share of total volume, an error line just adds noise rather than
// signal, so the chart skips drawing it entirely.
function isErrorMeaningful(valid: number[], error: number[]): boolean {
  const validSum = valid.reduce((a, b) => a + b, 0);
  const errorSum = error.reduce((a, b) => a + b, 0);
  const total = validSum + errorSum;
  return total > 0 && errorSum / total > 0.015;
}

// Andreas's full-bleed treatment: the curve is pinned to the row's bottom edge
// and kept faint enough that it reads as a backdrop, not a foreground chart.
// A background-tinted gradient over the top half guards the text/metric area
// above it in case a peak reaches up that far.
export function BleedRowChart({
  valid,
  error,
  className,
}: {
  valid: number[];
  error: number[];
  className?: string;
}) {
  const rawId = useId().replace(/[^a-zA-Z0-9]/g, "");
  if (valid.length === 0) {
    return null;
  }
  const w = 600;
  const h = 46;
  const validPath = buildSmoothPath(valid, w, h, 4);
  const errorPath = isErrorMeaningful(valid, error)
    ? buildScaledPath(error, w, h, 4, Math.max(...valid))
    : null;
  const fillId = `bleed-fill-${rawId}`;

  return (
    <svg viewBox={`0 0 ${w} ${h}`} className={className} preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id={fillId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={REQUEST_STROKE} stopOpacity="0.18" />
          <stop offset="100%" stopColor={REQUEST_STROKE} stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={validPath.area} fill={`url(#${fillId})`} />
      <path
        d={validPath.line}
        fill="none"
        stroke={REQUEST_STROKE}
        strokeOpacity={0.35}
        strokeWidth={1.5}
        vectorEffect="non-scaling-stroke"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {errorPath && (
        <path
          d={errorPath.line}
          fill="none"
          stroke={ERROR_STROKE}
          strokeOpacity={0.4}
          strokeWidth={1.25}
          vectorEffect="non-scaling-stroke"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      )}
    </svg>
  );
}

// The contained treatment: same curve quality, boxed to a fixed size that sits
// between the row's text and its metric, with a stronger fill since it isn't
// competing with text sitting on top of it.
export function ContainedRowChart({
  valid,
  error,
  className,
}: {
  valid: number[];
  error: number[];
  className?: string;
}) {
  const rawId = useId().replace(/[^a-zA-Z0-9]/g, "");
  if (valid.length === 0) {
    return <div className={className} />;
  }
  const w = 160;
  const h = 34;
  const validPath = buildSmoothPath(valid, w, h, 3);
  const errorPath = isErrorMeaningful(valid, error)
    ? buildScaledPath(error, w, h, 3, Math.max(...valid))
    : null;
  const fillId = `contained-fill-${rawId}`;

  return (
    <svg viewBox={`0 0 ${w} ${h}`} className={className} preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id={fillId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={REQUEST_STROKE} stopOpacity="0.35" />
          <stop offset="100%" stopColor={REQUEST_STROKE} stopOpacity="0.04" />
        </linearGradient>
      </defs>
      <path d={validPath.area} fill={`url(#${fillId})`} />
      <path
        d={validPath.line}
        fill="none"
        stroke={REQUEST_STROKE}
        strokeOpacity={0.7}
        strokeWidth={1.5}
        vectorEffect="non-scaling-stroke"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {errorPath && (
        <path
          d={errorPath.line}
          fill="none"
          stroke={ERROR_STROKE}
          strokeOpacity={0.6}
          strokeWidth={1.25}
          vectorEffect="non-scaling-stroke"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      )}
    </svg>
  );
}
