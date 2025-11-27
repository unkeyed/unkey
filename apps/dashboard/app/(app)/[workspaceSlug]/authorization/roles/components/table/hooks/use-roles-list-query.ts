import { useTRPC } from "@/lib/trpc/client";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { useEffect, useMemo, useState } from "react";
import { rolesFilterFieldConfig, rolesListFilterFieldNames } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { RolesQueryPayload } from "../query-logs.schema";

import { useInfiniteQuery } from "@tanstack/react-query";

export function useRolesListQuery() {
  const trpc = useTRPC();
  const [totalCount, setTotalCount] = useState(0);
  const [rolesMap, setRolesMap] = useState(() => new Map<string, RoleBasic>());
  const { filters } = useFilters();

  const rolesList = useMemo(() => Array.from(rolesMap.values()), [rolesMap]);

  const queryParams = useMemo(() => {
    const params: RolesQueryPayload = {
      ...Object.fromEntries(rolesListFilterFieldNames.map((field) => [field, []])),
    };

    filters.forEach((filter) => {
      if (!rolesListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        return;
      }

      const fieldConfig = rolesFilterFieldConfig[filter.field];
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
    data: rolesData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = useInfiniteQuery(
    trpc.authorization.roles.query.infiniteQueryOptions(queryParams, {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }),
  );

  useEffect(() => {
    if (rolesData) {
      const newMap = new Map<string, RoleBasic>();

      rolesData.pages.forEach((page) => {
        page.roles.forEach((role) => {
          // Use slug as the unique identifier
          newMap.set(role.roleId, role);
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
