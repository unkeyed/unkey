import { trpc } from "@/lib/trpc/client";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { useEffect, useMemo, useState } from "react";
import { rootKeysFilterFieldConfig, rootKeysListFilterFieldNames } from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/settings/root-keys/hooks/use-filters";
import type { RootKeysQueryPayload } from "../../schema/query-logs.schema";

export function useRootKeysListQuery() {
  const [totalCount, setTotalCount] = useState(0);
  const [rootKeysMap, setRootKeysMap] = useState(() => new Map<string, RootKey>());
  const { filters } = useFilters();

  const rootKeys = useMemo(() => Array.from(rootKeysMap.values()), [rootKeysMap]);

  const queryParams = useMemo(() => {
    const params: RootKeysQueryPayload = {
      ...Object.fromEntries(rootKeysListFilterFieldNames.map((field) => [field, []])),
    };

    filters.forEach((filter) => {
      if (!rootKeysListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        return;
      }

      const fieldConfig = rootKeysFilterFieldConfig[filter.field];
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
  }, [filters]);

  const {
    data: rootKeyData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.settings.rootKeys.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage: { nextCursor?: number }) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (rootKeyData) {
      const newMap = new Map<string, RootKey>();

      rootKeyData.pages.forEach((page: { keys: RootKey[] }) => {
        page.keys.forEach((rootKey: RootKey) => {
          // Use slug as the unique identifier
          newMap.set(rootKey.id, rootKey);
        });
      });

      if (rootKeyData.pages.length > 0) {
        setTotalCount(rootKeyData.pages[0].total);
      }

      setRootKeysMap(newMap);
    }
  }, [rootKeyData]);

  return {
    rootKeys,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    totalCount,
  };
}
