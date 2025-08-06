"use client";
import { trpc } from "@/lib/trpc/client";

import type {
  ApiOverview,
  ApisOverviewResponse,
} from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { type Dispatch, type SetStateAction, useEffect, useState } from "react";
import { DEFAULT_OVERVIEW_FETCH_LIMIT } from "../constants";

export const useFetchApiOverview = (
  initialData: ApisOverviewResponse,
  setApiList: Dispatch<SetStateAction<ApiOverview[]>>,
) => {
  const [hasMore, setHasMore] = useState(initialData.hasMore);
  const [cursor, setCursor] = useState(initialData.nextCursor);
  const [total, setTotal] = useState(initialData.total);
  const [isFetchingMore, setIsFetchingMore] = useState(false);

  const { data, isFetching, refetch } = trpc.api.overview.query.useQuery(
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
  }, [total, data, setApiList]);

  const loadMore = () => {
    if (!hasMore || isFetchingMore || !cursor) {
      return;
    }

    setIsFetchingMore(true);
    refetch();
  };

  const isLoading = isFetchingMore || isFetching;
  return { isLoading, total, loadMore, hasMore };
};

export function sortApisByKeyCount(apiOverview: ApiOverview[]) {
  return apiOverview.toSorted(
    (a, b) =>
      b.keys.reduce((acc, crr) => acc + crr.count, 0) -
      a.keys.reduce((acc, crr) => acc + crr.count, 0),
  );
}
