/**
 * This is a stand in for `portal.getVerifications`, so the analytics UI can
 * be reviewed without me wiring up the backend.
 *
 * THIS FILE IS TEMPORARY - DELETE IT WHEN YOURE DONE. `FAKE_ANALYTICS` switches both the analytics chart and the keys
 * table sparklines to generated data. This must not ship on: before merging,
 * set it to `null` (or delete this file and the two branches in
 * `portal-api.ts`) and wire the real timeseries.
 */

import {
  HOURLY_MAX_DAYS,
  type VerificationBucket,
} from "~/components/analytics/schema/analytics.schema";

export type FakeDataVariant = "populated" | "empty";

export const FAKE_ANALYTICS: FakeDataVariant | null = "populated";

const HOUR_MS = 3_600_000;
const DAY_MS = 86_400_000;
const KEY_USAGE_DAYS = 30;
const SEED = 1_337_042;

const HOUR_WEIGHTS = [
  0.28, 0.21, 0.16, 0.14, 0.15, 0.22, 0.38, 0.6, 0.85, 1.0, 1.06, 1.02, 0.9, 0.99, 1.09, 1.07, 0.96,
  0.81, 0.67, 0.59, 0.52, 0.46, 0.4, 0.33,
];
const DAY_WEIGHTS = [0.41, 1.0, 1.07, 1.05, 1.02, 0.94, 0.48];

function mulberry32(seed: number): () => number {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4_294_967_296;
  };
}

function hashString(value: string): number {
  let hash = 2_166_136_261;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16_777_619);
  }
  return hash >>> 0;
}

// The API fills from the bucket containing `start` through the bucket
// containing `end` inclusive, so a 24h window has 25 hourly buckets.
function bucketStarts(days: number): number[] {
  const hourly = days <= HOURLY_MAX_DAYS;
  const step = hourly ? HOUR_MS : DAY_MS;
  const count = (hourly ? 24 * days : days) + 1;
  const end = Math.floor(Date.now() / step) * step;
  const starts: number[] = [];
  for (let i = count - 1; i >= 0; i--) {
    starts.push(end - i * step);
  }
  return starts;
}

export function fakeVerifications(days: number, variant: FakeDataVariant): VerificationBucket[] {
  const starts = bucketStarts(days);
  if (variant === "empty") {
    return starts.map((time) => ({ time, total: 0, valid: 0, error: 0 }));
  }

  const random = mulberry32(SEED + days);
  const hourly = days <= HOURLY_MAX_DAYS;
  const base = hourly ? 1_450 : 27_500;

  return starts.map((time) => {
    const date = new Date(time);
    const rhythm = hourly ? HOUR_WEIGHTS[date.getHours()] : DAY_WEIGHTS[date.getDay()];
    const jitter = 0.82 + random() * 0.36;
    const spike = random() > 0.94 ? 1.6 + random() * 0.9 : 1;
    const total = Math.round(base * rhythm * jitter * spike);
    const burst = random() > 0.91 ? 2.8 + random() * 1.7 : 1;
    const rejected = Math.min(Math.round(total * 0.07 * (0.6 + random() * 0.8) * burst), total);
    return { time, total, valid: total - rejected, error: rejected };
  });
}

export function fakeKeyUsage(
  keyId: string,
  variant: FakeDataVariant,
): { usage: number[]; errors: number[] } {
  if (variant === "empty") {
    return {
      usage: new Array(KEY_USAGE_DAYS).fill(0),
      errors: new Array(KEY_USAGE_DAYS).fill(0),
    };
  }

  const hash = hashString(keyId);
  const random = mulberry32(SEED + hash);
  const perDay = 40 + (hash % 25_000);
  const errorRate = 0.02 + (hash % 11) / 100;
  const now = Math.floor(Date.now() / DAY_MS) * DAY_MS;

  const usage: number[] = [];
  const errors: number[] = [];
  for (let i = KEY_USAGE_DAYS - 1; i >= 0; i--) {
    const day = new Date(now - i * DAY_MS).getDay();
    const jitter = 0.7 + random() * 0.6;
    const spike = random() > 0.93 ? 2.1 + random() : 1;
    const total = Math.round(perDay * DAY_WEIGHTS[day] * jitter * spike);
    usage.push(total);
    errors.push(Math.min(Math.round(total * errorRate * (0.4 + random() * 1.6)), total));
  }
  return { usage, errors };
}
