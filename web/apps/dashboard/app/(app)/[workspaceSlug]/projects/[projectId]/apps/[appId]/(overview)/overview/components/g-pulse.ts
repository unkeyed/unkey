import type { AreaChartPoint } from "../../deployments/[deploymentId]/network/unkey-flow/components/overlay/node-details-panel/components/chart/area-timeseries-chart";

// SYNTHETIC pulse for the Option-G production card. Everything here is mock:
// the metrics backend is fixed to a 6h RPS window with no per-deployment 5xx
// split, so the adaptive window (hour/day/week) and the cumulative headline are
// modelled client-side until that lands. Values are deterministic (no
// Math.random / Date.now) so the card is hydration-stable.

export type WindowKey = "hour" | "day" | "week";
export type TrafficKey = "normal" | "zero";

export type Pulse = {
  windowKey: WindowKey;
  windowLabel: string;
  series: AreaChartPoint[];
  cumulative: number;
  rpsCurrent: number;
  errorsCurrent: number;
  latestTimestamp: number;
};

// Fixed "now" anchor (matches the older mock) so SSR and client render identical
// timestamps.
const NOW = 1_717_000_000_000;

const SPEC: Record<WindowKey, { points: number; stepMs: number; label: string }> = {
  hour: { points: 60, stepMs: 60_000, label: "this hour" },
  day: { points: 96, stepMs: 15 * 60_000, label: "today" },
  week: { points: 84, stepMs: 2 * 60 * 60_000, label: "this week" },
};

// Andreas's adaptive window: smallest step that still contains the app's age,
// capped at a week — so a fresh app shows "last hour" instead of a mostly-empty
// week, eliminating the coldstart emptiness.
export function adaptiveWindowForAge(ageMs: number): WindowKey {
  const HOUR = 60 * 60_000;
  const DAY = 24 * HOUR;
  if (ageMs < HOUR) {
    return "hour";
  }
  if (ageMs < DAY) {
    return "day";
  }
  return "week";
}

function noise(i: number, seed: number) {
  const x = Math.sin(i * 12.9898 + seed * 78.233) * 43758.5453;
  return x - Math.floor(x);
}
function jitter(i: number, seed: number) {
  return (noise(i, seed) - 0.5) * 1.4 + (noise(i * 0.5 + 3, seed + 11) - 0.5) * 0.6;
}
function gaussian(i: number, center: number, width: number) {
  const d = i - center;
  return Math.exp(-(d * d) / (2 * width * width));
}

function buildSeries(windowKey: WindowKey, traffic: TrafficKey) {
  const { points, stepMs } = SPEC[windowKey];
  const incident = Math.round(0.26 * (points - 1));
  const lateA = Math.round(0.8 * (points - 1));
  const lateB = Math.round(0.9 * (points - 1));

  return Array.from({ length: points }, (_, i) => {
    const originalTimestamp = NOW - (points - 1 - i) * stepMs;
    if (traffic === "zero") {
      // Coldstart: a near-flat floor of a few requests, no errors.
      const total = Math.max(0, Math.round((noise(i, 2) * 3 + jitter(i, 9)) * 10) / 10);
      return { originalTimestamp, total, errors: 0 };
    }
    const t = i / (points - 1);
    const level = t > 0.62 && t < 0.95 ? 90 : 16;
    const spike = gaussian(i, incident, 1.1) * 60;
    const blip = noise(i, 5) > 0.96 ? level * 1.3 : 0;
    const total = Math.max(
      2,
      Math.round((level * (1 + 0.2 * jitter(i, 1)) + spike + blip) * 10) / 10,
    );
    const burst = gaussian(i, incident, 1.8) * total * 0.4;
    const lateBursts = (gaussian(i, lateA, 1.5) * 0.16 + gaussian(i, lateB, 1.1) * 0.1) * total;
    const errors = Math.round(Math.min(total, burst + lateBursts) * 10) / 10;
    return { originalTimestamp, total, errors };
  });
}

export function buildPulse(windowKey: WindowKey, traffic: TrafficKey): Pulse {
  const { stepMs, label } = SPEC[windowKey];
  const series = buildSeries(windowKey, traffic);
  const bucketSeconds = stepMs / 1000;
  // Cumulative = Σ rps × bucket-seconds — the same derivation we'd use against
  // the live timeseries once it exists.
  const cumulative = Math.round(series.reduce((sum, p) => sum + (p.total ?? 0) * bucketSeconds, 0));
  const last = series.at(-1);
  return {
    windowKey,
    windowLabel: label,
    series,
    cumulative,
    rpsCurrent: last?.total ?? 0,
    errorsCurrent: last?.errors ?? 0,
    latestTimestamp: last?.originalTimestamp ?? NOW,
  };
}

// Compact count for the cumulative headline: 1240000 → "1.24M", 12400 → "12.4K".
export function formatCount(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(2).replace(/\.?0+$/, "")}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1).replace(/\.0$/, "")}K`;
  }
  return `${n}`;
}

export function formatRps(v: number): string {
  return `${v >= 10 ? Math.round(v) : Math.round(v * 10) / 10}/s`;
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

// The "as-of" stamp for the chart headline. Resolution follows the window, the
// same way the x-axis does: hour/day show a time; week shows a date at rest and
// a date + time while hovering a bucket. Keeps the stamp anchored to the axis
// below it instead of a bare time floating over a multi-day chart.
export function formatStamp(
  ts: number | undefined,
  windowKey: WindowKey,
  hovering: boolean,
): string {
  if (ts === undefined) {
    return "";
  }
  const d = new Date(ts);
  const time = `${String(d.getUTCHours()).padStart(2, "0")}:${String(d.getUTCMinutes()).padStart(2, "0")} UTC`;
  if (windowKey !== "week") {
    return time;
  }
  const date = `${d.getUTCDate()} ${MONTHS[d.getUTCMonth()]}`;
  return hovering ? `${date}, ${time}` : date;
}
