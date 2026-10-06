"use client";

import { trpc } from "@/lib/trpc/client";
import { useAppSettings } from "../settings";
import { type CrashHelp, resolveCrashHelp } from "./crash-help-state";

export function useCrashHelp(projectId: string, appId: string): CrashHelp {
  const settings = useAppSettings(projectId, appId);
  const { data: repoTree } = trpc.github.getRepoTree.useQuery(
    { projectId, appId },
    { staleTime: 5 * 60 * 1000 },
  );
  return resolveCrashHelp({
    settings: settings.status === "ready" ? settings.production : null,
    tree: repoTree?.tree ?? null,
  });
}
