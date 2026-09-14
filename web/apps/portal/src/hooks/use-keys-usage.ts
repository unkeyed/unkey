/**
 * The per-key stuff is a stand in for a grouped per-key verifications
 * endpoint the API does not offer yet. Everything the keys page fetches lives
 * here, so this is the one file to rewrite when that endpoint lands.
 */
import { type UseQueryResult, useQueries } from "@tanstack/react-query";
import { type KeyUsage, keyUsage } from "~/components/analytics/key-usage";
import type { VerificationBucket } from "~/components/analytics/schema/analytics.schema";
import type { TimeWindow } from "~/components/analytics/time-presets";
import type { Key } from "~/components/keys-table/schema/keys.schema";
import { useKeysListQuery } from "~/hooks/use-keys-list-query";
import { useVerificationsQuery, verificationsQueryOptions } from "~/hooks/use-verifications-query";
import { canReadAnalytics } from "~/lib/scopes";

// @dh warning - If we ship with this still here, the portal won't work properly.
const MAX_PER_KEY_USAGE_QUERIES = 50;

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
  /** "unavailable" when the session cannot read analytics or the account has more keys than we will fan out for. Otherwise one entry per key in `keys`, always. */
  byKey:
    | "unavailable"
    | { rows: ReadonlyMap<string, KeyUsageRow>; isPending: boolean; retryFailed: () => void };
};

function rowOf(key: Key, result: UseQueryResult<VerificationBucket[]>): KeyUsageRow {
  if (result.data) {
    return { key, status: "ok", usage: keyUsage(key, result.data) };
  }
  return { key, status: result.isError ? "error" : "pending" };
}

export function useKeysUsage(window: TimeWindow, scopes: ReadonlyArray<string>): KeysUsage {
  const analytics = canReadAnalytics(scopes);
  const keysList = useKeysListQuery();
  const aggregate = useVerificationsQuery(
    { startTime: window.startTime, endTime: window.endTime },
    analytics,
  );

  const fanOut = analytics && keysList.keys.length <= MAX_PER_KEY_USAGE_QUERIES;
  const queried = fanOut ? keysList.keys : [];

  const byKey = useQueries({
    queries: queried.map((key) =>
      verificationsQueryOptions({
        startTime: window.startTime,
        endTime: window.endTime,
        keyId: key.id,
      }),
    ),
    combine: (results) => ({
      rows: new Map(queried.map((key, i) => [key.id, rowOf(key, results[i])])),
      isPending: results.some((result) => result.isPending),
      retryFailed: () => {
        for (const result of results) {
          if (result.isError) {
            result.refetch();
          }
        }
      },
    }),
  });

  return {
    keys: keysList.keys,
    keysState: keysList,
    aggregate: aggregate.buckets,
    aggregateState: aggregate,
    analytics,
    byKey: fanOut ? byKey : "unavailable",
  };
}
