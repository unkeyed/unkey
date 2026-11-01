"use client";

import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { createContext, use, useEffect } from "react";

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
  awaitLiveDeployment: (target: LiveDeploymentTarget) => void;
  awaitDeploymentStatus: (target: DeploymentStatusTarget) => void;
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

export function useScopeStore(): ScopeStore {
  return useDeployDataContext().store;
}

export function createScopeStore() {
  let liveTarget: LiveDeploymentTarget | null = null;
  let statusTarget: DeploymentStatusTarget | null = null;
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
    statusTarget: () => statusTarget,
    requestedPollMs: () =>
      pollRequests.length === 0 ? null : Math.min(...pollRequests.map((r) => r.intervalMs)),
    setLiveTarget(target: LiveDeploymentTarget | null) {
      liveTarget = target;
      emit();
    },
    setStatusTarget(target: DeploymentStatusTarget | null) {
      statusTarget = target;
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
