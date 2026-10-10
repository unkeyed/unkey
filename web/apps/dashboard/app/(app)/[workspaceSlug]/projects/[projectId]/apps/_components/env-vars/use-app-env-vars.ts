import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";

export function useAppEnvVars(projectId: string, appId: string) {
  const { data, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );
  const { failed, retry } = useCollectionLoad(collection.envVars.utils);
  return { data, isLoading, failed, retry };
}
