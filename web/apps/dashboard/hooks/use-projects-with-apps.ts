"use client";
import { byLatestUpdate } from "@/lib/collections/deploy/project-cards";
import { useAppsOfProjects } from "@/lib/queries/apps";
import { useProjects } from "@/lib/queries/projects";

export function useProjectsWithApps({ enabled = true }: { enabled?: boolean } = {}) {
  const projects = useProjects({
    enabled,
    select: (all) => all.filter((p) => !p.isDefault),
  });
  const appQueries = useAppsOfProjects((projects.data ?? []).map((p) => p.id));
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
