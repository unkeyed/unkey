"use client";

import { refetchProjectApps } from "@/lib/collections/deploy/apps";
import { isDeploymentSettling } from "@/lib/collections/deploy/deployment-status";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { trpc } from "@/lib/trpc/client";
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useDeployScope, useScopeStore } from "./context";
import { deploymentStates, scopeDeployments, subscribeDeployments } from "./deployments";
import { deploymentsPollIntervalMs } from "./poll-interval";
import {
  refetchDeployData,
  refetchScopeCustomDomains,
  refetchScopeDeployments,
  refetchScopeDomains,
  useApp,
  useCustomDomains,
} from "./queries";

const AWAIT_TIMEOUT_MS = 60_000;
const POLL_APPS_IDLE_MS = 20_000;
const POLL_APPS_AWAITING_MS = 2_000;
const POLL_CUSTOM_DOMAINS_MS = 5_000;

// Deploys arrive from GitHub, the CLI and other people, so every app page
// refreshes the scope on a slow cadence. One instance per scope runs every poll.
export function DeployDataScheduler() {
  const { projectId, appId } = useDeployScope();
  const store = useScopeStore();
  const trpcUtils = trpc.useUtils();
  const { data: app } = useApp();
  const { data: customDomains } = useCustomDomains();

  const liveTarget = useSyncExternalStore(store.subscribe, store.liveTarget, store.liveTarget);
  const statusTarget = useSyncExternalStore(
    store.subscribe,
    store.statusTarget,
    store.statusTarget,
  );
  const requestedPollMs = useSyncExternalStore(
    store.subscribe,
    store.requestedPollMs,
    store.requestedPollMs,
  );
  const hasSettlingDeployment = useSyncExternalStore(
    subscribeDeployments,
    () => scopeDeployments(projectId, appId).some(isDeploymentSettling),
    () => false,
  );
  const statusReached = useSyncExternalStore(
    subscribeDeployments,
    () =>
      statusTarget !== null &&
      scopeDeployments(projectId, appId).some(
        (d) => d.id === statusTarget.deploymentId && d.status === statusTarget.status,
      ),
    () => false,
  );
  const liveReached =
    liveTarget !== null &&
    app?.currentDeploymentId === liveTarget.deploymentId &&
    app.isRolledBack === liveTarget.rolledBack;

  const settleLive = useCallback(() => {
    store.setLiveTarget(null);
    refetchDeployData(projectId, appId);
  }, [store, projectId, appId]);
  const settleStatus = useCallback(() => {
    store.setStatusTarget(null);
    refetchDeployData(projectId, appId);
  }, [store, projectId, appId]);
  useSettle(liveTarget, liveReached, settleLive);
  useSettle(statusTarget, statusReached, settleStatus);

  const [unchangedPolls, setUnchangedPolls] = useState(0);
  // An action of this tab starts a new burst of changes, so polling is fast again
  useEffect(() => {
    if (liveTarget !== null || statusTarget !== null) {
      setUnchangedPolls(0);
    }
  }, [liveTarget, statusTarget]);
  // Active branches only move when a deployment does
  const pollDeployments = useCallback(async () => {
    const before = deploymentStates(scopeDeployments(projectId, appId));
    await refetchScopeDeployments(projectId, appId);
    if (deploymentStates(scopeDeployments(projectId, appId)) !== before) {
      setUnchangedPolls(0);
      trpcUtils.deploy.deployment.listActiveBranches.invalidate();
    } else {
      setUnchangedPolls((count) => count + 1);
    }
  }, [trpcUtils, projectId, appId]);
  useCollectionPolling(pollDeployments, {
    intervalMs: deploymentsPollIntervalMs({
      awaiting: statusTarget !== null,
      settling: hasSettlingDeployment,
      requestedMs: requestedPollMs,
      unchangedPolls,
    }),
    enabled: true,
  });
  useCollectionPolling(() => refetchProjectApps(projectId), {
    intervalMs: liveTarget !== null ? POLL_APPS_AWAITING_MS : POLL_APPS_IDLE_MS,
    enabled: appId !== undefined,
  });
  useCollectionPolling(() => refetchScopeCustomDomains(projectId), {
    intervalMs: POLL_CUSTOM_DOMAINS_MS,
    enabled: customDomains.some(
      (d) => d.verificationStatus === "pending" || d.verificationStatus === "verifying",
    ),
  });

  // Domains move with the current deployment. Undefined means the app has not
  // loaded yet, so its first value is not a change.
  const currentDeploymentId = app === undefined ? undefined : app.currentDeploymentId;
  const previousDeploymentId = useRef(currentDeploymentId);
  useEffect(() => {
    const previous = previousDeploymentId.current;
    previousDeploymentId.current = currentDeploymentId;
    if (previous !== undefined && currentDeploymentId && previous !== currentDeploymentId) {
      refetchScopeDomains(projectId);
    }
  }, [currentDeploymentId, projectId]);

  return null;
}

// Settles a target once it is reached or the timeout passes
function useSettle(target: object | null, reached: boolean, settle: () => void) {
  useEffect(() => {
    if (target === null) {
      return;
    }
    if (reached) {
      settle();
      return;
    }
    const id = setTimeout(settle, AWAIT_TIMEOUT_MS);
    return () => clearTimeout(id);
  }, [target, reached, settle]);
}
