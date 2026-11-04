"use client";

import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { createContext, use, useEffect, useSyncExternalStore } from "react";

export type LiveDeploymentTarget = {
  deploymentId: string;
  rolledBack: boolean;
};

export type DeploymentStatusTarget = {
  deploymentId: string;
  status: DeploymentStatus;
};

export type DeployActions = {
  // Promote, rollback, stop and wake return before the backend has written the
  // state they cause. These record the state to wait for; polling speeds up
  // until it shows or a minute passes, then everything in scope refetches.
  // They resolve true once the state shows and false when the wait times out.
  awaitLiveDeployment: (target: LiveDeploymentTarget) => Promise<boolean>;
  awaitDeploymentStatus: (target: DeploymentStatusTarget) => Promise<boolean>;
  refetchDeployments: () => void;
  refetchApps: () => Promise<unknown>;
};

export type DeployScope = {
  projectId: string;
  // Absent on project pages (logs, requests), where reads cover every app
  appId: string | undefined;
};

type ScopeStore = ReturnType<typeof createScopeStore>;

type DeployDataContextValue = DeployScope & { actions: DeployActions; store: ScopeStore };

export const DeployDataContext = createContext<DeployDataContextValue | null>(null);

export function useDeployScope(): DeployScope {
  return useDeployDataContext();
}

export function useAppScope(): { projectId: string; appId: string } {
  const { projectId, appId } = useDeployDataContext();
  if (!appId) {
    throw new Error("useAppScope must be used inside an app-scoped (/apps/[appId]) route");
  }
  return { projectId, appId };
}

export function useDeployActions(): DeployActions {
  return useDeployDataContext().actions;
}

// Asks the scope's deployments poll to run at least this often while enabled
export function useDeploymentsPolling({
  intervalMs,
  enabled,
}: {
  intervalMs: number;
  enabled: boolean;
}): void {
  const { store } = useDeployDataContext();
  useEffect(
    () => (enabled ? store.requestPolling(intervalMs) : undefined),
    [store, intervalMs, enabled],
  );
}

// The deployment a promote or rollback is moving production traffic to, until it lands
export function useLiveTarget(): LiveDeploymentTarget | null {
  const { store } = useDeployDataContext();
  return useSyncExternalStore(store.subscribe, store.liveTarget, store.liveTarget);
}

// The status a stop, wake, cancel or approval is waiting for, until it shows
export function useAwaitedStatus(deploymentId: string): DeploymentStatus | undefined {
  const { store } = useDeployDataContext();
  return useSyncExternalStore(store.subscribe, store.statusTargets, store.statusTargets).get(
    deploymentId,
  );
}

export function useScopeStore(): ScopeStore {
  return useDeployDataContext().store;
}

export function createScopeStore() {
  let liveTarget: LiveDeploymentTarget | null = null;
  let liveWaiter: ((reached: boolean) => void) | null = null;
  // Keyed by deployment id, so awaiting one deployment does not drop another
  let statusTargets: ReadonlyMap<string, DeploymentStatus> = new Map();
  const statusWaiters = new Map<string, (reached: boolean) => void>();
  const pollRequests: Array<{ intervalMs: number }> = [];
  const listeners = new Set<() => void>();
  const emit = () => {
    for (const listener of listeners) {
      listener();
    }
  };

  return {
    subscribe(listener: () => void): () => void {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    liveTarget: () => liveTarget,
    statusTargets: () => statusTargets,
    requestedPollMs: () =>
      pollRequests.length === 0 ? null : Math.min(...pollRequests.map((r) => r.intervalMs)),
    awaitLive(target: LiveDeploymentTarget): Promise<boolean> {
      liveWaiter?.(false);
      liveTarget = target;
      emit();
      return new Promise((resolve) => {
        liveWaiter = resolve;
      });
    },
    settleLive(reached: boolean) {
      liveTarget = null;
      liveWaiter?.(reached);
      liveWaiter = null;
      emit();
    },
    awaitStatus({ deploymentId, status }: DeploymentStatusTarget): Promise<boolean> {
      statusWaiters.get(deploymentId)?.(false);
      statusTargets = new Map(statusTargets).set(deploymentId, status);
      emit();
      return new Promise((resolve) => {
        statusWaiters.set(deploymentId, resolve);
      });
    },
    settleStatus(deploymentId: string, reached: boolean) {
      const next = new Map(statusTargets);
      next.delete(deploymentId);
      statusTargets = next;
      statusWaiters.get(deploymentId)?.(reached);
      statusWaiters.delete(deploymentId);
      emit();
    },
    requestPolling(intervalMs: number): () => void {
      const request = { intervalMs };
      pollRequests.push(request);
      emit();
      return () => {
        pollRequests.splice(pollRequests.indexOf(request), 1);
        emit();
      };
    },
  };
}

function useDeployDataContext(): DeployDataContextValue {
  const context = use(DeployDataContext);
  if (!context) {
    throw new Error("Deploy data hooks must be used within DeployDataProvider");
  }
  return context;
}
