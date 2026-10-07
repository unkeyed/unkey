import { trpc } from "@/lib/trpc/client";
import { useCallback } from "react";
import { warmAppPage } from "../data-provider-queries";
import { ACTIVE_BRANCHES_PAGE_SIZE } from "../overview/components/active-branches/use-active-branches";

export function useWarmAppPage() {
  const trpcUtils = trpc.useUtils();
  return useCallback(
    (projectId: string, appId: string) => {
      warmAppPage(projectId, appId);
      trpcUtils.deploy.deployment.listActiveBranches.prefetchInfinite({
        projectId,
        appId,
        limit: ACTIVE_BRANCHES_PAGE_SIZE,
      });
    },
    [trpcUtils],
  );
}
