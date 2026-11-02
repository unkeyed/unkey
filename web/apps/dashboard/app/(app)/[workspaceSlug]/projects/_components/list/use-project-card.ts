"use client";
import { refetchProjectApps } from "@/lib/collections/deploy/apps";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { useLiveQuery } from "@tanstack/react-db";
import { projectAppsQueryFor } from "../../[projectId]/_components/apps-list/queries";

const IN_FLIGHT_POLL_MS = 5_000;

export function useProjectCard(projectId: string, { nearViewport }: { nearViewport: boolean }) {
  const projectApps = useLiveQuery(
    (q) => (nearViewport ? projectAppsQueryFor(projectId)(q) : null),
    [projectId, nearViewport],
  );
  const apps = projectApps.data ?? [];

  const inFlight = apps.some(
    (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
  );
  useCollectionPolling(() => refetchProjectApps(projectId), {
    intervalMs: IN_FLIGHT_POLL_MS,
    enabled: nearViewport && inFlight,
  });

  return { apps, isLoading: !nearViewport || projectApps.isLoading };
}
