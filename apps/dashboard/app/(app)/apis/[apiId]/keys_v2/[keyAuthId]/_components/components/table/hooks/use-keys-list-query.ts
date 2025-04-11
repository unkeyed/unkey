import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { useEffect, useMemo, useState } from "react";
import { keysListFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { KeysQueryListPayload } from "../query-logs.schema";

type UseKeysListQueryParams = {
  keyAuthId: string;
};

export function useKeysListQuery({ keyAuthId }: UseKeysListQueryParams) {
  const [totalCount, setTotalCount] = useState(0);
  const [keysMap, setKeysMap] = useState(() => new Map<string, KeyDetails>());

  const { filters } = useFilters();

  const keysList = useMemo(() => Array.from(keysMap.values()), [keysMap]);

  const queryParams = useMemo(() => {
    const params: KeysQueryListPayload = {
      keyIds: [],
      identities: [],
      names: [],
      keyAuthId,
    };

    filters.forEach((filter) => {
      const fieldConfig = keysListFilterFieldConfig[filter.field];
      const validOperators = fieldConfig.operators;
      const operator = validOperators.includes(filter.operator)
        ? filter.operator
        : validOperators[0];

      switch (filter.field) {
        case "keyIds": {
          if (typeof filter.value === "string") {
            const keyIdOperator = operator === "is" || operator === "contains" ? operator : "is";
            params.keyIds?.push({
              operator: keyIdOperator,
              value: filter.value,
            });
          }
          break;
        }
        case "names":
        case "identities":
          if (typeof filter.value === "string") {
            params[filter.field]?.push({
              operator,
              value: filter.value,
            });
          }
          break;
      }
    });

    return params;
  }, [filters, keyAuthId]);

  // Main query for keys data
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

  // Update keys map effect
  useEffect(() => {
    if (keysData) {
      const newMap = new Map<string, KeyDetails>();

      keysData.pages.forEach((page) => {
        page.keys.forEach((key) => {
          // Use key.id as the unique identifier
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
