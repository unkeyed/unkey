/**
 * Everything the keys page fetches. One `portal.getVerifications` call returns
 * every key's series, so the account-wide chart is their sum rather than a
 * second read, and the per-key table needs no fan-out.
 */
import { type KeyUsage, keyUsage, sumSeries } from "~/components/analytics/key-usage";
import type { KeySeries, VerificationBucket } from "~/components/analytics/schema/analytics.schema";
import type { TimeWindow } from "~/components/analytics/time-presets";
import type { Key } from "~/components/keys-table/schema/keys.schema";
import { useKeysListQuery } from "~/hooks/use-keys-list-query";
import { useVerificationsQuery } from "~/hooks/use-verifications-query";
import { canReadAnalytics } from "~/lib/scopes";

export type KeyUsageRow =
  | { key: Key; status: "pending" }
  | { key: Key; status: "error" }
  | { key: Key; status: "ok"; usage: KeyUsage };

type QueryState = {
  isInitialLoading: boolean;
  isError: boolean;
  error: unknown;
  refetch: () => void;
};

export type KeysUsage = {
  keys: Key[];
  keysState: QueryState;
  aggregate: VerificationBucket[];
  aggregateState: QueryState & { isFetching: boolean };
  /** Whether the session may read analytics at all; without it there are no numbers to show. */
  analytics: boolean;
  /** "unavailable" when the session cannot read analytics. Otherwise one entry per key in `keys`, always. */
  byKey:
    | "unavailable"
    | { rows: ReadonlyMap<string, KeyUsageRow>; isPending: boolean; retryFailed: () => void };
};

/**
 * Every row shares the one request's outcome. A key the response omits had no
 * traffic in the window, which is a settled zero rather than a missing number.
 */
function rowOf(key: Key, series: KeySeries | undefined, state: QueryState): KeyUsageRow {
  if (state.isError) {
    return { key, status: "error" };
  }
  if (state.isInitialLoading) {
    return { key, status: "pending" };
  }
  return { key, status: "ok", usage: keyUsage(key, series?.buckets ?? []) };
}

export function useKeysUsage(window: TimeWindow, scopes: ReadonlyArray<string>): KeysUsage {
  const analytics = canReadAnalytics(scopes);
  const keysList = useKeysListQuery();
  const verifications = useVerificationsQuery(
    { startTime: window.startTime, endTime: window.endTime },
    analytics,
  );

  // Series come from the verification events, so they can name a key the list
  // does not carry; the table is driven by the list and ignores those, while
  // the account-wide sum below deliberately keeps them.
  const seriesByKey = new Map(verifications.keys.map((series) => [series.keyId, series]));
  const rows = new Map(
    keysList.keys.map((key) => [key.id, rowOf(key, seriesByKey.get(key.id), verifications)]),
  );

  return {
    keys: keysList.keys,
    keysState: keysList,
    aggregate: sumSeries(verifications.keys),
    aggregateState: verifications,
    analytics,
    byKey: analytics
      ? {
          rows,
          isPending: verifications.isInitialLoading,
          retryFailed: verifications.refetch,
        }
      : "unavailable",
  };
}
