"use client";
import { trpc } from "@/lib/trpc/client";
import type { ApiOverview, ApisOverviewResponse } from "@/lib/trpc/routers/api/overview/schemas";
import { useEffect, useState } from "react";
import { DEFAULT_OVERVIEW_FETCH_LIMIT } from "../constants";

export const useFetchApiOverview = (initialData: ApisOverviewResponse) => {
  const [apiList, setApiList] = useState<ApiOverview[]>(initialData.apiList);
  const [hasMore, setHasMore] = useState(initialData.hasMore);
  const [cursor, setCursor] = useState(initialData.nextCursor);
  const [total, setTotal] = useState(initialData.total);
  const [isFetchingMore, setIsFetchingMore] = useState(false);

  const { data, isFetching, refetch } = trpc.api.overview.queryApisOverview.useQuery(
    { limit: DEFAULT_OVERVIEW_FETCH_LIMIT, cursor },
    {
      enabled: false,
    },
  );

  useEffect(() => {
    if (!data) {
      return;
    }

    const apisOrderedByKeyCount = sortApisByKeyCount(data.apiList);
    setApiList((prev) => [...prev, ...apisOrderedByKeyCount]);
    setHasMore(data.hasMore);
    setCursor(data.nextCursor);

    if (data.total !== total) {
      setTotal(data.total);
    }
    setIsFetchingMore(false);
  }, [total, data]);

  const loadMore = () => {
    if (!hasMore || isFetchingMore || !cursor) {
      return;
    }

    setIsFetchingMore(true);
    refetch();
  };

  const isLoading = isFetchingMore || isFetching;
  return { isLoading, total, loadMore, hasMore, apiList };
};

export function sortApisByKeyCount(apiOverview: ApiOverview[]) {
  return apiOverview.toSorted(
    (a, b) =>
      b.keys.reduce((acc, crr) => acc + crr.count, 0) -
      a.keys.reduce((acc, crr) => acc + crr.count, 0),
  );
}
