import { keepPreviousData, queryOptions, useQuery } from "@tanstack/react-query";
import type { GetVerificationsQuery } from "~/components/analytics/schema/analytics.schema";
import { getVerifications } from "~/lib/portal-api";

/**
 * One `v2/portal.getVerifications` call, keyed on the exact window and key so
 * every consumer asking for the same window shares a cache entry.
 */
function verificationsQueryOptions(query: GetVerificationsQuery) {
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
 * Loads the session end user's per-key verification timeseries for a window.
 * Keying on the window means a new selection fetches (and caches) a fresh
 * result without manual invalidation; the previous one stays on screen
 * meanwhile. `enabled` is false for a session without `analytics:read`, which
 * the API would refuse.
 */
export function useVerificationsQuery(query: GetVerificationsQuery, enabled = true) {
  const result = useQuery({ ...verificationsQueryOptions(query), enabled });

  return {
    keys: result.data ?? [],
    isInitialLoading: result.isLoading || (result.isFetching && !result.data),
    isFetching: result.isFetching,
    isError: result.isError,
    error: result.error,
    refetch: result.refetch,
  };
}
