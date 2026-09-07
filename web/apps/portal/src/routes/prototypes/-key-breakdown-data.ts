import type { VerificationBucket } from "~/components/analytics/schema/analytics.schema";
import { fakeVerifications } from "~/lib/fake-data";

export type KeyUsage = {
  id: string;
  name: string;
  start: string;
  enabled: boolean;
  buckets: VerificationBucket[];
  total: number;
  error: number;
};

const KEYS = [
  {
    id: "key_3f9kQmT2",
    name: "Production",
    start: "acme_3f9k",
    enabled: true,
    share: 0.73,
    errorBias: 0.8,
  },
  {
    id: "key_8xq2Lp0V",
    name: "Development",
    start: "acme_8xq2",
    enabled: true,
    share: 0.21,
    errorBias: 1.6,
  },
  {
    id: "key_v01cZr8N",
    name: "CI Pipeline",
    start: "acme_v01c",
    enabled: true,
    share: 0.06,
    errorBias: 1.1,
  },
  {
    id: "key_ld7mWe4H",
    name: "Legacy",
    start: "acme_ld7m",
    enabled: false,
    share: 0,
    errorBias: 0,
  },
];

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

// Splits every bucket of the aggregate series across the keys so that the
// per-key rows always sum back to the chart the end user is looking at.
export function fakeKeyBreakdown(days: number): { totals: VerificationBucket[]; keys: KeyUsage[] } {
  const totals = fakeVerifications(days, "populated");
  const random = mulberry32(9_001 + days);
  const perKey = KEYS.map(() => [] as VerificationBucket[]);

  for (const bucket of totals) {
    const weights = KEYS.map((k) => (k.share === 0 ? 0 : k.share * (0.8 + random() * 0.4)));
    const weightSum = weights.reduce((a, b) => a + b, 0) || 1;
    const errorWeights = KEYS.map((k, i) => weights[i] * k.errorBias);
    const errorSum = errorWeights.reduce((a, b) => a + b, 0) || 1;

    const taker = 0;
    let totalLeft = bucket.total;
    let errorLeft = bucket.error;
    const split = KEYS.map((_, i) => {
      if (i === taker) {
        return { total: 0, error: 0 };
      }
      const total = Math.round((bucket.total * weights[i]) / weightSum);
      const error = Math.min(Math.round((bucket.error * errorWeights[i]) / errorSum), total);
      totalLeft -= total;
      errorLeft -= error;
      return { total, error };
    });
    split[taker] = {
      total: Math.max(totalLeft, 0),
      error: Math.max(Math.min(errorLeft, totalLeft), 0),
    };
    split.forEach(({ total, error }, i) => {
      perKey[i].push({ time: bucket.time, total, valid: total - error, error });
    });
  }

  const keys = KEYS.map((k, i) => ({
    id: k.id,
    name: k.name,
    start: k.start,
    enabled: k.enabled,
    buckets: perKey[i],
    total: perKey[i].reduce((a, b) => a + b.total, 0),
    error: perKey[i].reduce((a, b) => a + b.error, 0),
  })).sort((a, b) => b.total - a.total);

  return { totals, keys };
}
