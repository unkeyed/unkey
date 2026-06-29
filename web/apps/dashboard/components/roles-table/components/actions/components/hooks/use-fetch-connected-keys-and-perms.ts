import { useRoleLimits } from "@/components/roles-table/hooks/use-role-limits";
import { trpc } from "@/lib/trpc/client";

export const useFetchConnectedKeysAndPermsData = (roleId: string) => {
  const { calculateLimits } = useRoleLimits(roleId);
  const { shouldPrefetch } = calculateLimits();

  const query = trpc.authorization.roles.connectedKeysAndPerms.useQuery(
    { roleId },
    {
      enabled: shouldPrefetch && Boolean(roleId),
      staleTime: 5 * 60 * 1000,
    },
  );

  return {
    keys: query.data?.keys || [],
    permissions: query.data?.permissions || [],
    hasData: Boolean(query.data),
    // Whether the connected keys/permissions query actually runs. It is disabled
    // for over-limit roles, so callers must not wait on `hasData` in that case
    // (the query never resolves and the data never arrives).
    shouldPrefetch,
    error: query.error,
  };
};
