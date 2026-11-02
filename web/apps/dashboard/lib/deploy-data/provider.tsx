"use client";

import { refetchProjectApps } from "@/lib/collections/deploy/apps";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { queryKeys } from "@/lib/query-keys";
import { trpc } from "@/lib/trpc/client";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import {
  type PropsWithChildren,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";
import { DeployDataContext, createScopeStore, useDeployScope, useScopeStore } from "./context";
import { deploymentDetailOptions } from "./deployments";
import {
  refetchDeployData,
  refetchScopeCustomDomains,
  refetchScopeDomains,
  useApp,
  useCustomDomains,
} from "./queries";

type DeployDataProviderProps = PropsWithChildren<{ projectId?: string; appId?: string }>;

// Scopes every deploy data hook below it to one project, and to one app when
// there is one. The context value changes only when the scope does.
export function DeployDataProvider({
  children,
  projectId: projectIdProp,
  appId: appIdProp,
}: DeployDataProviderProps) {
  const params = useParams();
  const projectId =
    projectIdProp ?? (typeof params?.projectId === "string" ? params.projectId : undefined);
  const appId = appIdProp ?? (typeof params?.appId === "string" ? params.appId : undefined);
  if (!projectId) {
    throw new Error("DeployDataProvider requires a projectId prop or a [projectId] route param");
  }

  const trpcUtils = trpc.useUtils();
  const queryClient = useQueryClient();
  const value = useMemo(() => {
    const store = createScopeStore();
    return {
      projectId,
      appId,
      store,
      actions: {
        awaitLiveDeployment: store.setLiveTarget,
        awaitDeploymentStatus: store.addStatusTarget,
        refetchDeployments: () => {
          queryClient.invalidateQueries({
            queryKey: queryKeys.deployments.scope(projectId),
          });
          trpcUtils.deploy.deployment.listActiveBranches.invalidate();
          trpcUtils.deploy.deployment.listBranches.invalidate();
        },
        refetchApps: () => refetchProjectApps(projectId),
      },
    };
  }, [projectId, appId, trpcUtils, queryClient]);

  return (
    <DeployDataContext.Provider value={value}>
      <DeployDataScheduler />
      {children}
    </DeployDataContext.Provider>
  );
}

const AWAIT_TIMEOUT_MS = 60_000;
const POLL_AWAITING_MS = 2_000;
const POLL_APPS_IDLE_MS = 20_000;
const POLL_CUSTOM_DOMAINS_MS = 5_000;

// Deploys arrive from GitHub, the CLI and other people, so app pages refresh
// apps and pending custom domains on a slow cadence; deployments queries poll on
// their own. One instance per scope also waits out the actions of this tab.
function DeployDataScheduler() {
  const { projectId, appId } = useDeployScope();
  const store = useScopeStore();
  const queryClient = useQueryClient();
  const { data: app } = useApp();
  const { data: customDomains = [] } = useCustomDomains();

  const liveTarget = useSyncExternalStore(store.subscribe, store.liveTarget, store.liveTarget);
  const statusTargets = useSyncExternalStore(
    store.subscribe,
    store.statusTargets,
    store.statusTargets,
  );
  const liveReached =
    liveTarget !== null &&
    app?.currentDeploymentId === liveTarget.deploymentId &&
    app.isRolledBack === liveTarget.rolledBack;

  const refetchAll = useCallback(() => {
    refetchDeployData(projectId);
    queryClient.invalidateQueries({ queryKey: queryKeys.deployments.scope(projectId) });
  }, [queryClient, projectId]);
  const settleLive = useCallback(() => {
    store.setLiveTarget(null);
    refetchAll();
  }, [store, refetchAll]);
  const settleStatus = useCallback(
    (deploymentId: string) => {
      store.removeStatusTarget(deploymentId);
      refetchAll();
    },
    [store, refetchAll],
  );
  useSettle(liveTarget, liveReached, settleLive);

  useCollectionPolling(() => refetchProjectApps(projectId), {
    intervalMs: liveTarget !== null ? POLL_AWAITING_MS : POLL_APPS_IDLE_MS,
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
  const currentDeploymentId = app === undefined ? undefined : (app?.currentDeploymentId ?? null);
  const previousDeploymentId = useRef(currentDeploymentId);
  useEffect(() => {
    const previous = previousDeploymentId.current;
    previousDeploymentId.current = currentDeploymentId;
    if (previous !== undefined && currentDeploymentId && previous !== currentDeploymentId) {
      refetchScopeDomains(projectId);
    }
  }, [currentDeploymentId, projectId]);

  return [...statusTargets].map(([deploymentId, status]) => (
    <AwaitedStatus
      key={deploymentId}
      deploymentId={deploymentId}
      status={status}
      onSettled={settleStatus}
    />
  ));
}

function AwaitedStatus({
  deploymentId,
  status,
  onSettled,
}: {
  deploymentId: string;
  status: DeploymentStatus;
  onSettled: (deploymentId: string) => void;
}) {
  const { projectId } = useDeployScope();
  const queryClient = useQueryClient();
  const { data } = useQuery({
    ...deploymentDetailOptions(queryClient, projectId, deploymentId),
    refetchInterval: POLL_AWAITING_MS,
  });
  const settle = useCallback(() => onSettled(deploymentId), [onSettled, deploymentId]);
  useSettle(status, data?.status === status, settle);
  return null;
}

// Settles a target once it is reached or the timeout passes
function useSettle(target: unknown, reached: boolean, settle: () => void) {
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
