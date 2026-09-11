import { keepPreviousData, queryOptions, useQuery } from "@tanstack/react-query";
import type { GetVerificationsQuery } from "~/components/analytics/schema/analytics.schema";
import { getVerifications } from "~/lib/portal-api";

/**
 * One `v2/portal.getVerifications` call, keyed on the exact window and key so
 * the analytics chart, the per-key table and the keys-page sparklines share a
 * cache entry whenever they ask for the same series.
 */
export function verificationsQueryOptions(query: GetVerificationsQuery) {
  return queryOptions({
    queryKey: [
      "portal",
      "analytics",
      "verifications",
      query.startTime,
      query.endTime,
      query.keyId ?? null,
    ] as const,
    queryFn: () => getVerifications({ data: query }),
    placeholderData: keepPreviousData,
    staleTime: 1000 * 60,
    refetchOnWindowFocus: false,
  });
}

/**
 * Loads the session end user's verification timeseries for a window. Keying
 * on the window means a new selection fetches (and caches) a fresh series
 * without manual invalidation; the previous series stays on screen meanwhile.
 * `enabled` is false for a session without `analytics:read`, which the API
 * would refuse.
 */
export function useVerificationsQuery(query: GetVerificationsQuery, enabled = true) {
  const result = useQuery({ ...verificationsQueryOptions(query), enabled });

  return {
    buckets: result.data ?? [],
    isInitialLoading: result.isLoading || (result.isFetching && !result.data),
    isFetching: result.isFetching,
    isError: result.isError,
    error: result.error,
    refetch: result.refetch,
  };
}
