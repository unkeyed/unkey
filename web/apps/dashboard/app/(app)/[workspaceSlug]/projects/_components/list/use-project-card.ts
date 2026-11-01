"use client";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { refetchProjectApps, useAppsInProject } from "@/lib/deploy-data";

const IN_FLIGHT_POLL_MS = 5_000;

export function useProjectCard(projectId: string, { nearViewport }: { nearViewport: boolean }) {
  const enabled = nearViewport && projectId !== SERVER_PLACEHOLDER;

  const projectApps = useAppsInProject(projectId, { enabled });
  const apps = projectApps.data;

  const inFlight = apps.some(
    (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
  );
  useCollectionPolling(() => refetchProjectApps(projectId), {
    intervalMs: IN_FLIGHT_POLL_MS,
    enabled: enabled && inFlight,
  });

  return { apps, isLoading: !nearViewport || projectApps.isLoading };
}
