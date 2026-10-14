import { trpc } from "@/lib/trpc/client";
import { useCallback, useEffect, useRef } from "react";
import { warmAppPage } from "../data-provider-queries";
import { ACTIVE_BRANCHES_PAGE_SIZE } from "../overview/components/active-branches/use-active-branches";

// A pointer passing over a list of apps should not warm each one it crosses
const WARM_DELAY_MS = 150;

export function useWarmAppPage() {
  const trpcUtils = trpc.useUtils();
  const pendingWarm = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const cancelWarm = useCallback(() => clearTimeout(pendingWarm.current), []);
  useEffect(() => cancelWarm, [cancelWarm]);

  const warmApp = useCallback(
    (projectId: string, appId: string) => {
      clearTimeout(pendingWarm.current);
      pendingWarm.current = setTimeout(() => {
        warmAppPage(projectId, appId);
        trpcUtils.deploy.deployment.listActiveBranches.prefetchInfinite({
          projectId,
          appId,
          limit: ACTIVE_BRANCHES_PAGE_SIZE,
        });
      }, WARM_DELAY_MS);
    },
    [trpcUtils],
  );

  return { warmApp, cancelWarm };
}
