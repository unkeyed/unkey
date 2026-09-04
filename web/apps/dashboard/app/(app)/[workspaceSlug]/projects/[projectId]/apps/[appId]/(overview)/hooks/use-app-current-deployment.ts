import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useAppId, useProjectData } from "../data-provider";

// The current deployment can be older than the newest-100 window the provider
// holds, so it is resolved by id. The collection loads a single-id subset on
// demand.
export function useAppCurrentDeployment() {
  const { projectId } = useProjectData();
  const appId = useAppId();

  const appsQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appsQuery.data?.[0];
  const currentDeploymentId = app?.currentDeploymentId ?? null;

  const currentDeploymentQuery = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) =>
          and(
            eq(deployment.projectId, projectId),
            eq(deployment.appId, appId),
            eq(deployment.id, currentDeploymentId ?? ""),
          ),
        ),
    [projectId, appId, currentDeploymentId],
  );

  return {
    app,
    currentDeployment: currentDeploymentId ? currentDeploymentQuery.data?.[0] : undefined,
    isRolledBack: app?.isRolledBack ?? false,
    isLoading:
      appsQuery.isLoading || (currentDeploymentId !== null && currentDeploymentQuery.isLoading),
  };
}
