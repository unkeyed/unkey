"use client";
import { useAppsEnvironments } from "@/lib/queries/environments";

/** Environments of every app in the given projects. Idle until `projects` loads. */
export function useProjectEnvironments(
  projects: ReadonlyArray<{ id: string; apps: ReadonlyArray<{ id: string }> }> | undefined,
) {
  return useAppsEnvironments(
    (projects ?? []).flatMap((project) =>
      project.apps.map((app) => ({ projectId: project.id, appId: app.id })),
    ),
  );
}
