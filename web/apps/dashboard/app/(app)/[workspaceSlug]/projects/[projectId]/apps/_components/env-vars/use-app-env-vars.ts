import { collection } from "@/lib/collections";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";

export type AppEnvVarsLoad =
  | { status: "loading" }
  | { status: "failed"; retry: () => void }
  | { status: "ready"; data: EnvVar[] };

export function useAppEnvVars(projectId: string, appId: string): AppEnvVarsLoad {
  const { data, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );
  const { failed, retry } = useCollectionLoad(collection.envVars.utils);
  if (failed) {
    return { status: "failed", retry };
  }
  if (isLoading) {
    return { status: "loading" };
  }
  return { status: "ready", data };
}
