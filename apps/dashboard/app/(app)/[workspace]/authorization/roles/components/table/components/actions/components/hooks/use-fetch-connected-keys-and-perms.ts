import { useRoleLimits } from "@/app/(app)/[workspace]/authorization/roles/components/table/hooks/use-role-limits";
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
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
};
