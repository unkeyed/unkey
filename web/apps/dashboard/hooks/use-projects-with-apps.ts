"use client";
import { byLatestUpdate } from "@/lib/collections/deploy/project-cards";
import { projectAppsOptions } from "@/lib/queries/apps";
import { useProjects } from "@/lib/queries/projects";
import { useQueries } from "@tanstack/react-query";

export function useProjectsWithApps({ enabled = true }: { enabled?: boolean } = {}) {
  const projects = useProjects({
    enabled,
    select: (all) => all.filter((p) => !p.isDefault),
  });
  const appQueries = useQueries({
    queries: (projects.data ?? []).map((p) => projectAppsOptions(p.id)),
  });
  const appsByProject = Map.groupBy(
    appQueries.flatMap((query) => query.data ?? []),
    (app) => app.projectId,
  );

  return {
    data: projects.data
      ?.toSorted((a, b) => b.createdAt - a.createdAt)
      .map((project) => ({
        id: project.id,
        name: project.name,
        apps: (appsByProject.get(project.id) ?? [])
          .toSorted(byLatestUpdate)
          .map((app) => ({ id: app.id, name: app.name })),
      })),
    isLoading: projects.isInitialLoading || appQueries.some((query) => query.isInitialLoading),
    isError: projects.isError || appQueries.some((query) => query.isError),
  };
}
