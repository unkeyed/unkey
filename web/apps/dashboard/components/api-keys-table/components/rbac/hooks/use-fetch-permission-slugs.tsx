"use client";
import { queryKeys } from "@/lib/query-keys";

import { getUnkeyClient } from "@/lib/unkey-client";
import { useQueries } from "@tanstack/react-query";

export const keysRbacRoleQueryOptions = (roleName: string) => ({
  queryKey: queryKeys.rbac.roles.detail(roleName),
  queryFn: () => getUnkeyClient().permissions.getRole({ role: roleName }),
  staleTime: 30_000,
});

export const useFetchPermissionSlugs = (
  roleNames: string[] = [],
  directPermissionSlugs: string[] = [],
  enabled = true,
) => {
  const roleQueries = useQueries({
    queries: Array.from(new Set(roleNames)).map((roleName) => ({
      ...keysRbacRoleQueryOptions(roleName),
      enabled,
    })),
  });

  const isLoading = roleQueries.some((query) => query.isLoading);
  const hasError = roleQueries.some((query) => query.isError);

  if (!enabled || isLoading || hasError) {
    return { data: undefined, isLoading };
  }

  const slugs = new Set(directPermissionSlugs);
  for (const query of roleQueries) {
    for (const permission of query.data?.data.permissions ?? []) {
      slugs.add(permission.slug);
    }
  }
  const sortedSlugs = Array.from(slugs).sort();

  return {
    data: { slugs: sortedSlugs, totalCount: sortedSlugs.length },
    isLoading,
  };
};
