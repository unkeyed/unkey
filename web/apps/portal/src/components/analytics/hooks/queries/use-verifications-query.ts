import { useQuery } from "@tanstack/react-query";
import { getVerifications } from "~/lib/portal-api";
import type { VerificationBucket } from "../../schema/analytics.schema";

/** Shared query key prefix for the portal verification analytics. */
export const verificationsQueryKey = (days: number) =>
  ["portal", "analytics", "verifications", days] as const;

/**
 * Loads the session end user's verification timeseries via
 * `v2/portal.getVerifications` for the selected window (in days). Keying the
 * query on `days` means changing the selector fetches (and caches) a fresh
 * window without manual invalidation.
 *
 * Returns a small, purpose-built interface rather than the raw query object,
 * matching the keys page's `useKeysListQuery` convention.
 */
export function useVerificationsQuery(days: number) {
  const query = useQuery({
    queryKey: verificationsQueryKey(days),
    queryFn: () => getVerifications({ data: { days } }),
    staleTime: 1000 * 60, // 1 minute
    refetchOnWindowFocus: false,
  });

  const buckets: VerificationBucket[] = query.data?.buckets ?? [];

  return {
    buckets,
    // Initial load: no data yet. Distinct from a background refetch.
    isInitialLoading: query.isLoading || (query.isFetching && !query.data),
    isFetching: query.isFetching,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}
