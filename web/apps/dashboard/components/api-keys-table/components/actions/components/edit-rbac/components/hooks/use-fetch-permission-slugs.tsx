"use client";
import { trpc } from "@/lib/trpc/client";

export const useFetchPermissionSlugs = (
  roleNames: string[] = [],
  directPermissionSlugs: string[] = [],
  enabled = true,
) => {
  const { data, isLoading, error, refetch } = trpc.key.queryPermissionSlugs.useQuery(
    {
      roleNames,
      permissionSlugs: directPermissionSlugs,
    },
    {
      enabled,
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    },
  );

  return {
    data,
    isLoading,
    error,
    refetch,
    hasData: !isLoading && data !== undefined,
  };
};
