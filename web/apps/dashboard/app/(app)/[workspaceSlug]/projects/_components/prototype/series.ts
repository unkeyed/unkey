import type { Bucket } from "./marks";
import { hashCode, mulberry32 } from "./store";

const HOURS_IN_DAY = 24;

// Hourly {valid, error} pairs shaped like real traffic: a business-hours hump,
// per-point noise, rare short spikes, and errors as a small share of valid
// traffic with occasional blips. `points[i]` is `points - 1 - i` hours ago.
export function projectRequestSeries(
  seedKey: string,
  points: number,
  baseMagnitude: number,
): Bucket[] {
  const rand = mulberry32(hashCode(seedKey));
  const series: Bucket[] = [];

  const diurnalMultiplier = (i: number) => {
    const hourOfDay = (points - 1 - i) % 24;
    const phase = ((hourOfDay - 14) / 24) * Math.PI * 2;
    return 0.35 + 0.65 * (0.5 + 0.5 * Math.cos(phase));
  };

  let spikeCooldown = 0;

  for (let i = 0; i < points; i++) {
    const noise = 0.85 + rand() * 0.3;
    let magnitude = baseMagnitude * diurnalMultiplier(i) * noise;

    if (spikeCooldown > 0) {
      spikeCooldown--;
    } else if (rand() < 0.03) {
      magnitude *= 1.8 + rand() * 1.4;
      spikeCooldown = 1 + Math.floor(rand() * 2);
    }

    const valid = Math.max(0, Math.round(magnitude));
    const errorBlip = rand() < 0.05;
    const errorRate = errorBlip ? 0.04 + rand() * 0.08 : 0.005 + rand() * 0.035;
    const error = Math.max(0, Math.round(valid * errorRate));

    series.push({ valid, error });
  }

  return series;
}

// A resource's last 24h as hourly buckets, seeded so the same keyspace draws
// the same chart wherever it appears. `total` is derived from the buckets rather
// than passed through, so a hover readout can never contradict the row total.
export function hourlySeries(
  seed: string,
  total24h: number,
  errorPct: number,
): { buckets: Bucket[]; totals: number[]; total: number } {
  const pct = Math.min(100, Math.max(0, errorPct)) / 100;
  const raw = projectRequestSeries(seed, HOURS_IN_DAY, Math.max(1, total24h / HOURS_IN_DAY));
  const volume = raw.reduce((sum, b) => sum + b.valid + b.error, 0);
  const generatedErrors = raw.reduce((sum, b) => sum + b.error, 0) || 1;
  // Scale the generator's error series so its total lands on the resource's real
  // invalid/blocked rate. Re-deriving each bucket from the flat rate instead
  // paints an identical orange cap on every bar, which is the giveaway that no
  // real error data was involved.
  const scale = (volume * pct) / generatedErrors;
  const buckets = raw.map(({ valid, error }) => {
    const total = valid + error;
    const errored = Math.min(total, Math.round(error * scale));
    return { valid: Math.max(0, total - errored), error: errored };
  });
  return {
    buckets,
    totals: buckets.map((b) => b.valid + b.error),
    total: buckets.reduce((sum, b) => sum + b.valid + b.error, 0),
  };
}

// Seeds a keyspace's chart the same way on every screen it appears on.
export function keyspaceSeries(keyspace: {
  id: string;
  requests: { "24h": number };
  validPct: number;
}) {
  return hourlySeries(`keyspace-${keyspace.id}`, keyspace.requests["24h"], 100 - keyspace.validPct);
}

export function ratelimitSeries(ratelimit: {
  id: string;
  checks: { "24h": number };
  blockedPct: number;
}) {
  return hourlySeries(`ratelimit-${ratelimit.id}`, ratelimit.checks["24h"], ratelimit.blockedPct);
}
