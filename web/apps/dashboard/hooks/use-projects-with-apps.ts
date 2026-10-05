"use client";
import { collection } from "@/lib/collections";
import { byLatestUpdate } from "@/lib/collections/deploy/project-cards";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";

export function useProjectsWithApps({ enabled = true }: { enabled?: boolean } = {}) {
  const projects = useLiveQuery(
    (q) =>
      enabled
        ? q
            .from({ project: collection.projects })
            .where(({ project }) => eq(project.isDefault, false))
        : null,
    [enabled],
  );
  const projectIds = (projects.data ?? [])
    .map((p) => p.id)
    .filter((id) => id !== SERVER_PLACEHOLDER)
    .sort();
  const apps = useLiveQuery(
    (q) =>
      projectIds.length === 0
        ? null
        : q.from({ app: collection.apps }).where(({ app }) => inArray(app.projectId, projectIds)),
    [projectIds.join(",")],
  );

  const data = useMemo(() => {
    if (!projects.data || projects.isLoading) {
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
  }, [projects.data, projects.isLoading, apps.data]);

  return {
    data,
    isLoading: projects.isLoading || apps.isLoading,
    isError: projects.isError || apps.isError,
  };
}
