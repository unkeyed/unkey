import { trpc } from "@/lib/trpc/client";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { type Dispatch, type SetStateAction, useEffect, useState } from "react";
import { DEFAULT_OVERVIEW_FETCH_LIMIT } from "../constants";

export const useFetchApiOverview = (setApiList: Dispatch<SetStateAction<ApiOverview[]>>) => {
  const [hasMore, setHasMore] = useState<boolean>(true);
  const [cursor, setCursor] = useState<{ id: string } | undefined>();
  const [total, setTotal] = useState<number>();
  const [isFetchingMore, setIsFetchingMore] = useState(false);

  const { data, isFetching, refetch, isError, error } = trpc.api.overview.query.useQuery({
    limit: DEFAULT_OVERVIEW_FETCH_LIMIT,
    cursor,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
  useEffect(() => {
    if (!data) {
      if (!isFetching) {
        setIsFetchingMore(false);
      }
      return;
    }

    const apisOrderedByKeyCount = sortApisByKeyCount(data.apiList);

    setApiList((prev) => {
      const existingIds = new Set(prev.map((api) => api.id));
      const newUniqueApis = apisOrderedByKeyCount.filter((api) => !existingIds.has(api.id));
      return [...prev, ...newUniqueApis];
    });
    setHasMore(data.hasMore);
    setCursor(data.nextCursor);

    if (data.total !== total) {
      setTotal(data.total);
    }
    setIsFetchingMore(false);
  }, [data, isFetching, setApiList]);

  useEffect(() => {
    if (isError) {
      console.error("Failed to fetch API overview:", error);
      setIsFetchingMore(false);
    }
  }, [isError, error]);

  const loadMore = () => {
    if (!hasMore || isFetchingMore || !cursor) {
      return;
    }

    setIsFetchingMore(true);
    refetch();
  };

  const isLoading = isFetching;

  return { isLoading: isLoading, total, loadMore, hasMore };
};

export function sortApisByKeyCount(apiOverview: ApiOverview[]) {
  if (!Array.isArray(apiOverview)) {
    console.warn("sortApisByKeyCount received non-array:", apiOverview);
    return [];
  }
  return [...apiOverview].sort(
    (a, b) =>
      b.keys.reduce((acc, crr) => acc + crr.count, 0) -
      a.keys.reduce((acc, crr) => acc + crr.count, 0),
  );
}
