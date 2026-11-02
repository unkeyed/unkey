"use client";
import { byLatestUpdate } from "@/lib/collections/deploy/project-cards";
import { hasInFlightDeployment, useProjectApps } from "@/lib/queries/apps";

const IN_FLIGHT_POLL_MS = 5_000;
const IDLE_POLL_MS = 60_000;

export function useProjectCard(projectId: string, { nearViewport }: { nearViewport: boolean }) {
  const projectApps = useProjectApps(projectId, {
    enabled: nearViewport,
    refetchInterval: (apps) => (hasInFlightDeployment(apps) ? IN_FLIGHT_POLL_MS : IDLE_POLL_MS),
  });
  return {
    apps: (projectApps.data ?? []).toSorted(byLatestUpdate),
    isLoading: !nearViewport || projectApps.isInitialLoading,
  };
}
