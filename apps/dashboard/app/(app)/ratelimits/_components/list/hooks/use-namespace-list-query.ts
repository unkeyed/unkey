import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";

import { useNamespaceListFilters } from "../../hooks/use-namespace-list-filters";
import {
  namespaceListFilterFieldConfig,
  namespaceListFilterFieldNames,
} from "../../namespace-list-filters.schema";
import type { NamespaceListInputSchema, RatelimitNamespace } from "../namespace-list.schema";

export function useNamespaceListQuery() {
  const [totalCount, setTotalCount] = useState(0);
  const [namespacesMap, setNamespacesMap] = useState(() => new Map<string, RatelimitNamespace>());
  const { filters } = useNamespaceListFilters();

  const namespaces = useMemo(() => Array.from(namespacesMap.values()), [namespacesMap]);

  const queryParams = useMemo(() => {
    const params: NamespaceListInputSchema = {
      ...Object.fromEntries(namespaceListFilterFieldNames.map((field) => [field, []])),
    };

    filters.forEach((filter) => {
      if (!namespaceListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        return;
      }

      const fieldConfig = namespaceListFilterFieldConfig[filter.field];
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
    data: namespaceData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.ratelimit.namespace.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (namespaceData) {
      const newMap = new Map<string, RatelimitNamespace>();
      namespaceData.pages.forEach((page) => {
        page.namespaceList.forEach((namespace) => {
          newMap.set(namespace.id, namespace);
        });
      });

      if (namespaceData.pages.length > 0) {
        setTotalCount(namespaceData.pages[0].total);
      }

      setNamespacesMap(newMap);
    }
  }, [namespaceData]);

  return {
    namespaces,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    totalCount,
  };
}
