"use client";

import { refetchProjectApps } from "@/lib/collections/deploy/apps";
import { trpc } from "@/lib/trpc/client";
import { useParams } from "next/navigation";
import { type PropsWithChildren, useMemo } from "react";
import { DeployDataContext, createScopeStore } from "./context";
import { refetchScopeDeployments } from "./queries";
import { DeployDataScheduler } from "./scheduler";

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
  const value = useMemo(() => {
    const store = createScopeStore();
    return {
      projectId,
      appId,
      store,
      actions: {
        awaitLiveDeployment: store.setLiveTarget,
        awaitDeploymentStatus: store.setStatusTarget,
        refetchDeployments: () => {
          refetchScopeDeployments(projectId, appId);
          trpcUtils.deploy.deployment.listActiveBranches.invalidate();
          trpcUtils.deploy.deployment.listBranches.invalidate();
        },
        refetchApps: () => refetchProjectApps(projectId),
      },
    };
  }, [projectId, appId, trpcUtils]);

  return (
    <DeployDataContext.Provider value={value}>
      <DeployDataScheduler />
      {children}
    </DeployDataContext.Provider>
  );
}
