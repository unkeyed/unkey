"use client";
import { useTRPC } from "@/lib/trpc/client";

import { useQuery } from "@tanstack/react-query";

export const useFetchPermissionSlugs = (
  roleIds: string[] = [],
  directPermissionIds: string[] = [],
  enabled = true,
) => {
  const trpc = useTRPC();
  const { data, isLoading, error, refetch } = useQuery(
    trpc.key.queryPermissionSlugs.queryOptions(
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
    ),
  );

  return {
    data,
    isLoading,
    error,
    refetch,
    hasData: !isLoading && data !== undefined,
  };
};
