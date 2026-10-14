import { collection } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useSyncExternalStore } from "react";

export type AppEnvVarsLoad =
  | { status: "loading" }
  | { status: "failed"; retry: () => void }
  | { status: "ready"; data: EnvVar[]; refreshing: boolean };

export function useAppEnvVars(projectId: string, appId: string): AppEnvVarsLoad {
  const { data, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );
  const { failed, retry } = useCollectionLoad(collection.envVars.utils);
  const refreshing = useSyncExternalStore(
    (onChange) => queryClient.getQueryCache().subscribe(onChange),
    () => queryClient.isFetching({ queryKey: ["envVars", appId] }) > 0,
    () => true,
  );
  if (failed) {
    return { status: "failed", retry };
  }
  if (isLoading) {
    return { status: "loading" };
  }
  return { status: "ready", data, refreshing };
}
