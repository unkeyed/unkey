import { useInfiniteQuery } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { listKeys } from "~/lib/portal-api";
import type { Key } from "../../schema/keys.schema";

/** Shared query key for the portal keys list, so mutations can invalidate it. */
export const keysListQueryKey = ["portal", "keys", "list"] as const;

const PAGE_SIZE = 100;

/**
 * Loads the session end user's keys via `v2/portal.listKeys`, following the
 * response cursor across pages and accumulating them into a single list. The
 * keys-table then does search, filter, sort, and pagination client-side over
 * the full set (the same model the design prototype uses).
 *
 * Returns a small, purpose-built interface rather than the raw query object,
 * matching the dashboard's `useApiKeysListQuery` convention.
 */
export function useKeysListQuery() {
  const query = useInfiniteQuery({
    queryKey: keysListQueryKey,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => listKeys({ data: { cursor: pageParam, limit: PAGE_SIZE } }),
    getNextPageParam: (lastPage) => (lastPage.hasMore ? (lastPage.cursor ?? undefined) : undefined),
    staleTime: 1000 * 60, // 1 minute
    refetchOnWindowFocus: false,
  });

  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;

  // Eagerly pull remaining pages so client-side search/filter/sort operate over
  // the complete set. Portal end users typically have few keys.
  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const keys = useMemo<Key[]>(
    () => query.data?.pages.flatMap((page) => page.keys) ?? [],
    [query.data],
  );

  return {
    keys,
    // Initial load: no data yet. Distinct from background refetch/pagination.
    isInitialLoading: query.isLoading || (query.isFetching && !query.data),
    isFetching: query.isFetching || isFetchingNextPage,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}
