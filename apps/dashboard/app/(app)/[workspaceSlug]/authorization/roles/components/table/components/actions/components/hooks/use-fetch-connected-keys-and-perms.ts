import { useRoleLimits } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/table/hooks/use-role-limits";
import { useTRPC } from "@/lib/trpc/client";

import { useQuery } from "@tanstack/react-query";

export const useFetchConnectedKeysAndPermsData = (roleId: string) => {
  const trpc = useTRPC();
  const { calculateLimits } = useRoleLimits(roleId);
  const { shouldPrefetch } = calculateLimits();

  const query = useQuery(trpc.authorization.roles.connectedKeysAndPerms.queryOptions(
    { roleId },
    {
      enabled: shouldPrefetch && Boolean(roleId),
      staleTime: 5 * 60 * 1000,
    },
  ));

  return {
    keys: query.data?.keys || [],
    permissions: query.data?.permissions || [],
    hasData: Boolean(query.data),
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
};
