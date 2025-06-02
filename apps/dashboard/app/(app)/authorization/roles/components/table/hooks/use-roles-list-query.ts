import { trpc } from "@/lib/trpc/client";
import type { Roles } from "@/lib/trpc/routers/authorization/roles";
import { useEffect, useMemo, useState } from "react";
import { keysListFilterFieldNames, rolesFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";

type RolesQueryPayload = {
  limit?: number;
  cursor?: number;
};

export function useRolesListQuery() {
  const [totalCount, setTotalCount] = useState(0);
  const [rolesMap, setRolesMap] = useState(() => new Map<string, Roles>());
  const { filters } = useFilters();

  const rolesList = useMemo(() => Array.from(rolesMap.values()), [rolesMap]);

  // For now, roles endpoint doesn't support filtering, so we prepare for future enhancement
  const queryParams = useMemo((): RolesQueryPayload => {
    // Currently the roles endpoint only supports limit and cursor
    // Filter validation is kept for future compatibility
    filters.forEach((filter) => {
      if (!keysListFilterFieldNames.includes(filter.field)) {
        throw new Error(`Invalid filter field: ${filter.field}`);
      }

      const fieldConfig = rolesFilterFieldConfig[filter.field];
      if (!fieldConfig.operators.includes(filter.operator)) {
        throw new Error(`Invalid operator '${filter.operator}' for field '${filter.field}'`);
      }

      if (typeof filter.value !== "string") {
        throw new Error(`Filter value must be a string for field '${filter.field}'`);
      }
    });

    return {
      limit: 50, // Using default from your endpoint
    };
  }, [filters]);

  const {
    data: rolesData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.authorization.roles.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (rolesData) {
      const newMap = new Map<string, Roles>();

      rolesData.pages.forEach((page) => {
        page.roles.forEach((role) => {
          // Use slug as the unique identifier
          newMap.set(role.slug, role);
        });
      });

      if (rolesData.pages.length > 0) {
        setTotalCount(rolesData.pages[0].total);
      }

      setRolesMap(newMap);
    }
  }, [rolesData]);

  return {
    roles: rolesList,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    totalCount,
  };
}
