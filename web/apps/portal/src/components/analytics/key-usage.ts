import type { Key } from "~/components/keys-table/schema/keys.schema";
import {
  type VerificationTotals,
  emptyBucket,
  emptyOutcomes,
  sumBuckets,
} from "./analytics-transform";
import type { OutcomeFilterKind } from "./outcomes";
import { OUTCOME_KINDS, type VerificationBucket } from "./schema/analytics.schema";

/** One key's series over the selected window, plus its window totals. */
export type KeyUsage = VerificationTotals & {
  key: Key;
  buckets: VerificationBucket[];
};

export function keyUsage(key: Key, buckets: VerificationBucket[]): KeyUsage {
  return { key, buckets, ...sumBuckets(buckets) };
}

/** The series a set of keys add up to, for a chart narrowed to some of them. */
export function sumSeries(keys: ReadonlyArray<KeyUsage>): VerificationBucket[] {
  const byTime = new Map<number, VerificationBucket>();
  for (const usage of keys) {
    for (const bucket of usage.buckets) {
      const sum = byTime.get(bucket.time) ?? emptyBucket(bucket.time);
      sum.total += bucket.total;
      sum.valid += bucket.valid;
      sum.error += bucket.error;
      sum.other += bucket.other;
      for (const kind of OUTCOME_KINDS) {
        sum[kind] += bucket[kind];
      }
      byTime.set(bucket.time, sum);
    }
  }
  return [...byTime.values()].sort((a, b) => a.time - b.time);
}

function projectBucket(
  bucket: VerificationBucket,
  outcomes: ReadonlyArray<OutcomeFilterKind>,
): VerificationBucket {
  const kept = emptyOutcomes();
  let error = 0;
  for (const kind of OUTCOME_KINDS) {
    if (outcomes.includes(kind)) {
      kept[kind] = bucket[kind];
      error += bucket[kind];
    }
  }
  const valid = outcomes.includes("valid") ? bucket.valid : 0;
  return { ...kept, time: bucket.time, total: valid + error, valid, error, other: 0 };
}

function facetCountsOf(totals: VerificationTotals): Record<OutcomeFilterKind, number> {
  const {
    valid,
    rateLimited,
    expired,
    disabled,
    usageExceeded,
    insufficientPermissions,
    forbidden,
  } = totals;
  return {
    valid,
    rateLimited,
    expired,
    disabled,
    usageExceeded,
    insufficientPermissions,
    forbidden,
  };
}

export type FilteredUsage = {
  keys: KeyUsage[];
  totals: VerificationBucket[];
  /** Per-outcome totals over the unprojected series, for the facet counts. */
  facetCounts: Record<OutcomeFilterKind, number>;
};

/**
 * An outcome filter narrows the chart and the table together, as on the
 * dashboard. Keys emptied by it drop out; with no filter set a key with no
 * traffic stays visible, so "unused" is still a finding. Projection is linear,
 * so `series` is projected in place rather than rebuilt from `keys` — a series
 * the caller took from the account aggregate stays complete even when the
 * per-key numbers are short or absent.
 */
export function applyFilters(
  keys: ReadonlyArray<KeyUsage>,
  series: VerificationBucket[],
  outcomes: ReadonlyArray<OutcomeFilterKind>,
): FilteredUsage {
  const facetCounts = facetCountsOf(sumBuckets(series));

  if (outcomes.length === 0) {
    return { keys: [...keys], totals: series, facetCounts };
  }

  return {
    keys: keys
      .map((entry) =>
        keyUsage(
          entry.key,
          entry.buckets.map((bucket) => projectBucket(bucket, outcomes)),
        ),
      )
      .filter((entry) => entry.total > 0),
    totals: series.map((bucket) => projectBucket(bucket, outcomes)),
    facetCounts,
  };
}
