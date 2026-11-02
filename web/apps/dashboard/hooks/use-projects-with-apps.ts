"use client";
import { collection } from "@/lib/collections";
import { byLatestUpdate } from "@/lib/collections/deploy/project-cards";
import { useProjects } from "@/lib/queries/projects";
import { inArray, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";

export function useProjectsWithApps({ enabled = true }: { enabled?: boolean } = {}) {
  const projects = useProjects({
    enabled,
    select: (all) => all.filter((p) => !p.isDefault),
  });
  const projectIds = (projects.data ?? []).map((p) => p.id).sort();
  const apps = useLiveQuery(
    (q) =>
      projectIds.length === 0
        ? null
        : q.from({ app: collection.apps }).where(({ app }) => inArray(app.projectId, projectIds)),
    [projectIds.join(",")],
  );

  const data = useMemo(() => {
    if (!projects.data) {
      return undefined;
    }
    const appsByProject = Map.groupBy(apps.data ?? [], (app) => app.projectId);
    return projects.data
      .toSorted((a, b) => b.createdAt - a.createdAt)
      .map((project) => ({
        id: project.id,
        name: project.name,
        apps: (appsByProject.get(project.id) ?? [])
          .toSorted(byLatestUpdate)
          .map((app) => ({ id: app.id, name: app.name })),
      }));
  }, [projects.data, apps.data]);

  return {
    data,
    isLoading: projects.isLoading || apps.isLoading,
    isError: projects.isError || apps.isError,
  };
}
