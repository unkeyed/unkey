import { queryKeys } from "@/lib/query-keys";
import { trpc } from "@/lib/trpc/client";
import { useQueryClient } from "@tanstack/react-query";

export const useInvalidateRbacQueries = () => {
  const trpcUtils = trpc.useUtils();
  const queryClient = useQueryClient();

  return () =>
    Promise.all([
      // Paginated lists never refetch on mount, so refetch them while unmounted too
      trpcUtils.authorization.roles.query.invalidate(undefined, { refetchType: "all" }),
      trpcUtils.authorization.permissions.query.invalidate(undefined, { refetchType: "all" }),
      trpcUtils.api.keys.list.invalidate(undefined, { refetchType: "all" }),
      trpcUtils.authorization.roles.connectedKeysAndPerms.invalidate(),
      trpcUtils.authorization.roles.keys.invalidate(),
      trpcUtils.authorization.roles.permissions.invalidate(),
      trpcUtils.key.connectedRolesAndPerms.invalidate(),
      queryClient.invalidateQueries({ queryKey: queryKeys.rbac.all }),
      queryClient.invalidateQueries({ queryKey: queryKeys.keys.all }),
    ]);
};
