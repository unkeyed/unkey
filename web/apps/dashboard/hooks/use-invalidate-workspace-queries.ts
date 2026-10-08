import { queryKeys } from "@/lib/query-keys";
import { trpc } from "@/lib/trpc/client";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

export const useInvalidateWorkspaceQueries = () => {
  const trpcUtils = trpc.useUtils();
  const queryClient = useQueryClient();

  return useCallback(
    () =>
      Promise.all([
        trpcUtils.workspace.getCurrent.invalidate(),
        queryClient.invalidateQueries({ queryKey: queryKeys.workspace.all }),
      ]),
    [trpcUtils, queryClient],
  );
};
