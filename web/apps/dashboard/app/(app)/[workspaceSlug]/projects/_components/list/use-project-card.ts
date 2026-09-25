"use client";
import { queryClient } from "@/lib/collections/client";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { useLiveQuery } from "@tanstack/react-db";
import { projectAppsQueryFor } from "../../[projectId]/_components/apps-list/queries";

const IN_FLIGHT_POLL_MS = 5_000;

export function useProjectCard(projectId: string, { nearViewport }: { nearViewport: boolean }) {
  const enabled = nearViewport && projectId !== SERVER_PLACEHOLDER;

  const projectApps = useLiveQuery(
    (q) => (enabled ? projectAppsQueryFor(projectId)(q) : null),
    [projectId, enabled],
  );
  const apps = projectApps.data ?? [];

  const inFlight = apps.some(
    (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
  );
  useCollectionPolling(
    () => queryClient.refetchQueries({ queryKey: ["apps", projectId], type: "active" }),
    { intervalMs: IN_FLIGHT_POLL_MS, enabled: enabled && inFlight },
  );

  return { apps, isLoading: !nearViewport || projectApps.isLoading };
}
