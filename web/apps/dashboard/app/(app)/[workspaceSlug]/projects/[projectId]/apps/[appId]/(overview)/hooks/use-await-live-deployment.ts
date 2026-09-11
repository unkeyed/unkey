import { collection } from "@/lib/collections";
import type { App } from "@/lib/collections/deploy/apps";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { useCallback, useEffect, useState } from "react";

const POLL_INTERVAL_MS = 2_000;
const TIMEOUT_MS = 60_000;

export type LiveDeploymentTarget = {
  deploymentId: string;
  rolledBack: boolean;
};

// Promote, rollback and undo-rollback return 202 and swap the live deployment
// in a Restate workflow after the response, so a single refetch reads the old
// row. The returned function takes the state the app should reach; the hook
// polls the apps collection until the app reports it (or a minute passes),
// then runs onSettled to refresh every cache that shows the live deployment.
export function useAwaitLiveDeployment(
  app: App | undefined,
  onSettled: () => void,
): (target: LiveDeploymentTarget) => void {
  const [target, setTarget] = useState<LiveDeploymentTarget | null>(null);

  const settle = useCallback(() => {
    setTarget(null);
    onSettled();
  }, [onSettled]);

  useCollectionPolling(() => collection.apps.utils.refetch(), {
    intervalMs: POLL_INTERVAL_MS,
    enabled: target !== null,
  });

  const reached =
    target !== null &&
    app?.currentDeploymentId === target.deploymentId &&
    app.isRolledBack === target.rolledBack;

  useEffect(() => {
    if (reached) {
      settle();
    }
  }, [reached, settle]);

  useEffect(() => {
    if (target === null) {
      return;
    }
    const id = setTimeout(settle, TIMEOUT_MS);
    return () => clearTimeout(id);
  }, [target, settle]);

  return useCallback((next: LiveDeploymentTarget) => {
    setTarget(next);
    collection.apps.utils.refetch();
  }, []);
}
