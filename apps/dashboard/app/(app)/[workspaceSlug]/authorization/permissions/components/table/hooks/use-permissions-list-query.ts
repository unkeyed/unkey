import { useTRPC } from "@/lib/trpc/client";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { useEffect, useMemo, useState } from "react";
import {
  permissionsFilterFieldConfig,
  permissionsListFilterFieldNames,
} from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { PermissionsQueryPayload } from "../query-logs.schema";

import { useInfiniteQuery } from "@tanstack/react-query";

export function usePermissionsListQuery() {
  const trpc = useTRPC();
  const [totalCount, setTotalCount] = useState(0);
  const [permissionsMap, setPermissionsMap] = useState(() => new Map<string, Permission>());
  const { filters } = useFilters();

  const permissionsList = useMemo(() => Array.from(permissionsMap.values()), [permissionsMap]);

  const queryParams = useMemo(() => {
    const params: PermissionsQueryPayload = {
      ...Object.fromEntries(permissionsListFilterFieldNames.map((field) => [field, []])),
    };

    filters.forEach((filter) => {
      if (!permissionsListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        return;
      }

      const fieldConfig = permissionsFilterFieldConfig[filter.field];
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
    data: permissionsData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = useInfiniteQuery(
    trpc.authorization.permissions.query.infiniteQueryOptions(queryParams, {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }),
  );

  useEffect(() => {
    if (permissionsData) {
      const newMap = new Map<string, Permission>();

      permissionsData.pages.forEach((page) => {
        page.permissions.forEach((permission) => {
          // Use permissionId as the unique identifier
          newMap.set(permission.permissionId, permission);
        });
      });

      if (permissionsData.pages.length > 0) {
        setTotalCount(permissionsData.pages[0].total);
      }

      setPermissionsMap(newMap);
    }
  }, [permissionsData]);

  return {
    permissions: permissionsList,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    totalCount,
  };
}
