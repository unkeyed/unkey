"use client";
import { appsPollInterval, useProjectApps } from "@/lib/queries/apps";

export function useProjectCard(projectId: string, { nearViewport }: { nearViewport: boolean }) {
  const projectApps = useProjectApps(projectId, {
    enabled: nearViewport,
    refetchInterval: appsPollInterval,
  });
  return {
    apps: projectApps.data ?? [],
    isLoading: !nearViewport || projectApps.isInitialLoading,
  };
}
