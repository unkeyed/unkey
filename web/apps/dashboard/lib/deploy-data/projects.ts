"use client";

import { collection } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import {
  type Environment,
  type ProjectApps,
  inProjectApps,
} from "@/lib/collections/deploy/environments";
import { byLatestUpdate } from "@/lib/collections/deploy/project-cards";
import type { Project } from "@/lib/collections/deploy/projects";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { createLiveQueryCollection, inArray, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { type QueryResult, memoByArgs, useLoadState } from "./queries";

// These hooks read the workspace's projects and need no DeployDataProvider

export type ProjectWithApps = {
  id: string;
  name: string;
  apps: Array<{ id: string; name: string }>;
};

// Every project of the workspace, newest first
export function useProjects({
  enabled = true,
}: { enabled?: boolean } = {}): QueryResult<Project[]> {
  const query = useLiveQuery(() => (enabled ? projectsQuery() : null), [enabled]);
  const load = useLoadState(collection.projects.utils, refetchProjects, "");
  return { data: query.data ?? [], isLoading: query.isLoading, ...load };
}

export function useProject(projectId: string | undefined): QueryResult<Project | undefined> {
  const projects = useProjects({ enabled: projectId !== undefined });
  return { ...projects, data: projects.data.find((project) => project.id === projectId) };
}

// The named projects (not the workspace default) with their apps. Undefined until projects load.
export function useProjectsWithApps({
  enabled = true,
}: { enabled?: boolean } = {}): QueryResult<ProjectWithApps[] | undefined> {
  const projects = useProjects({ enabled });
  const projectIds = projects.data
    .filter((project) => !project.isDefault)
    .map((project) => project.id)
    .filter((id) => id !== SERVER_PLACEHOLDER)
    .sort()
    .join(",");
  const apps = useLiveQuery(
    (q) =>
      projectIds === ""
        ? null
        : q
            .from({ app: collection.apps })
            .where(({ app }) => inArray(app.projectId, projectIds.split(","))),
    [projectIds],
  );

  const projectsData = projects.data;
  const appsData = apps.data;
  const data = useMemo(() => {
    if (!enabled || projects.isLoading) {
      return undefined;
    }
    const appsByProject = Map.groupBy(appsData ?? [], (app) => app.projectId);
    return projectsData
      .filter((project) => !project.isDefault)
      .map((project) => ({
        id: project.id,
        name: project.name,
        apps: (appsByProject.get(project.id) ?? [])
          .toSorted(byLatestUpdate)
          .map((app) => ({ id: app.id, name: app.name })),
      }));
  }, [enabled, projects.isLoading, projectsData, appsData]);

  return {
    ...projects,
    data,
    isLoading: projects.isLoading || apps.isLoading,
    isError: projects.isError || apps.isError,
  };
}

// The environments of every app in the given projects. Idle until `projects` loads.
export function useProjectEnvironments(
  projects: ReadonlyArray<ProjectApps> | undefined,
): QueryResult<Environment[]> {
  const withApps = (projects ?? []).filter((project) => project.apps.length > 0);
  const scopeKey = withApps
    .map((project) => `${project.id}:${project.apps.map((app) => app.id).join(",")}`)
    .join(";");

  const query = useLiveQuery(
    (q) =>
      withApps.length === 0
        ? null
        : q.from({ env: collection.environments }).where(({ env }) => inProjectApps(env, withApps)),
    [scopeKey],
  );
  const load = useLoadState(collection.environments.utils, refetchEnvironments, "");
  return { data: query.data ?? [], isLoading: query.isLoading, ...load };
}

export function refetchProjects() {
  return collection.projects.utils.refetch();
}

const projectsQuery = memoByArgs(() =>
  createLiveQueryCollection((q) =>
    q.from({ project: collection.projects }).orderBy(({ project }) => project.createdAt, "desc"),
  ),
);

function refetchEnvironments() {
  return queryClient.refetchQueries({ queryKey: ["environments"], type: "active" });
}
