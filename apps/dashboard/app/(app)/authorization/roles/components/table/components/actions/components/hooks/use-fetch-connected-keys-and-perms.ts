"use client";
import { trpc } from "@/lib/trpc/client";

export const useFetchConnectedKeysAndPerms = (roleId: string) => {
  const { data, isLoading, error, refetch } =
    trpc.authorization.roles.connectedKeysAndPerms.useQuery(
      {
        roleId,
      },
      {
        enabled: Boolean(roleId),
      },
    );

  return {
    keys: data?.keys || [],
    permissions: data?.permissions || [],
    isLoading,
    error,
    refetch,
  };
};
