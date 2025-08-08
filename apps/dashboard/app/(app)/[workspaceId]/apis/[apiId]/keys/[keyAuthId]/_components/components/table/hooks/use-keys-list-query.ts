import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { useEffect, useMemo, useState } from "react";
import { keysListFilterFieldConfig, keysListFilterFieldNames } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { KeysQueryListPayload } from "../query-logs.schema";

type UseKeysListQueryParams = {
  keyAuthId: string;
};

const CURSOR_LIMIT = 50;
export function useKeysListQuery({ keyAuthId }: UseKeysListQueryParams) {
  const [totalCount, setTotalCount] = useState(0);
  const [keysMap, setKeysMap] = useState(() => new Map<string, KeyDetails>());

  const { filters } = useFilters();

  const keysList = useMemo(() => Array.from(keysMap.values()), [keysMap]);

  const queryParams = useMemo(() => {
    const params: KeysQueryListPayload = {
      limit: CURSOR_LIMIT,
      ...Object.fromEntries(keysListFilterFieldNames.map((field) => [field, []])),
      keyAuthId,
    };

    filters.forEach((filter) => {
      if (!keysListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        return;
      }

      const fieldConfig = keysListFilterFieldConfig[filter.field];
      const validOperators = fieldConfig.operators;
      if (!validOperators.includes(filter.operator)) {
        throw new Error("Invalid operator");
      }

      if (typeof filter.value === "string") {
        params[filter.field]?.push({
          operator: filter.operator,
          value: filter.value,
        });
      }
    });
    return params;
  }, [filters, keyAuthId]);

  const {
    data: keysData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.api.keys.list.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (keysData) {
      const newMap = new Map<string, KeyDetails>();
      keysData.pages.forEach((page) => {
        page.keys.forEach((key) => {
          newMap.set(key.id, key);
        });
      });
      if (keysData.pages.length > 0) {
        setTotalCount(keysData.pages[0].totalCount);
      }
      setKeysMap(newMap);
    }
  }, [keysData]);

  return {
    keys: keysList,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    totalCount,
  };
}
