"use client";
import { trpc } from "@/lib/trpc/client";

export const useFetchPermissionSlugs = (
  roleIds: string[] = [],
  directPermissionIds: string[] = [],
  enabled = true,
) => {
  const { data, isLoading, error, refetch } = trpc.key.queryPermissionSlugs.useQuery(
    {
      roleIds,
      permissionIds: directPermissionIds,
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
