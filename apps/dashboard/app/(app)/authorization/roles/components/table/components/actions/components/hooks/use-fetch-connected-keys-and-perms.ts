import { trpc } from "@/lib/trpc/client";

export const useFetchConnectedKeysAndPermsData = (roleId: string, shouldFetch = true) => {
  const query = trpc.authorization.roles.connectedKeysAndPerms.useQuery(
    { roleId },
    {
      enabled: shouldFetch && Boolean(roleId),
      staleTime: 5 * 60 * 1000,
    },
  );

  return {
    keys: query.data?.keys || [],
    permissions: query.data?.permissions || [],
    hasData: Boolean(query.data),
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
};
