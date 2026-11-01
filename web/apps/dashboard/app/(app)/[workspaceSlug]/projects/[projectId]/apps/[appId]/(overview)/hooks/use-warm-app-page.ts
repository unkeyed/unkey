import { useAppScope, warmAppPage, warmDeployment } from "@/lib/deploy-data";
import { trpc } from "@/lib/trpc/client";
import { useCallback, useEffect, useRef } from "react";
import { ACTIVE_BRANCHES_PAGE_SIZE } from "../overview/components/active-branches/use-active-branches";

// A pointer passing over a list should not warm each row it crosses
const WARM_DELAY_MS = 150;

export function useWarmAppPage() {
  const trpcUtils = trpc.useUtils();
  const { schedule, cancel } = useDelayedWarm();

  const warmApp = useCallback(
    (projectId: string, appId: string) =>
      schedule(() => {
        warmAppPage(projectId, appId);
        trpcUtils.deploy.deployment.listActiveBranches.prefetchInfinite({
          projectId,
          appId,
          limit: ACTIVE_BRANCHES_PAGE_SIZE,
        });
      }),
    [schedule, trpcUtils],
  );

  return { warmApp, cancelWarm: cancel };
}

export function useWarmAppHandlers(projectId: string, appId: string): WarmHandlers {
  const { warmApp, cancelWarm } = useWarmAppPage();
  const warm = () => warmApp(projectId, appId);
  return { onPointerEnter: warm, onPointerLeave: cancelWarm, onFocusCapture: warm };
}

export function useWarmDeploymentHandlers(deploymentId: string): WarmHandlers {
  const { projectId, appId } = useAppScope();
  const trpcUtils = trpc.useUtils();
  const { schedule, cancel } = useDelayedWarm();
  const warm = () =>
    schedule(() => {
      warmDeployment(projectId, appId, deploymentId);
      trpcUtils.deploy.deployment.steps.prefetch({ deploymentId });
    });
  return { onPointerEnter: warm, onPointerLeave: cancel, onFocusCapture: warm };
}

function useDelayedWarm() {
  const pending = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const cancel = useCallback(() => clearTimeout(pending.current), []);
  useEffect(() => cancel, [cancel]);
  const schedule = useCallback((warm: () => void) => {
    clearTimeout(pending.current);
    pending.current = setTimeout(warm, WARM_DELAY_MS);
  }, []);
  return { schedule, cancel };
}

export type WarmHandlers = {
  onPointerEnter: () => void;
  onPointerLeave: () => void;
  onFocusCapture: () => void;
};
