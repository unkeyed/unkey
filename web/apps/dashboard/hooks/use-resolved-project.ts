"use client";

import { collection } from "@/lib/collections";
import type { Project } from "@/lib/collections/deploy/projects";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";

/**
 * Resolves the [projectSlug] route param to the project row (and id) via the
 * projects collection. For consumers outside ProjectDataProvider; inside it,
 * use useProjectData() instead.
 */
export function useResolvedProject(): {
  projectSlug: string | undefined;
  projectId: string | undefined;
  project: Project | undefined;
  isLoading: boolean;
} {
  const params = useParams();
  const projectSlug = typeof params?.projectSlug === "string" ? params.projectSlug : undefined;

  const projectQuery = useLiveQuery(
    (q) =>
      q
        .from({ project: collection.projects })
        .where(({ project }) => eq(project.slug, projectSlug ?? "")),
    [projectSlug],
  );
  const project = projectSlug ? projectQuery.data?.at(0) : undefined;

  return {
    projectSlug,
    projectId: project?.id,
    project,
    isLoading: projectQuery.isLoading,
  };
}

/**
 * Resolves an app slug to its app id within the current [projectSlug] route.
 * App slugs are only unique per project, so this waits for the project id.
 */
export function useResolvedApp(appSlug: string | undefined): { appId: string | undefined } {
  const { projectId } = useResolvedProject();

  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId ?? ""), eq(app.slug, appSlug ?? ""))),
    [projectId, appSlug],
  );
  const app = projectId && appSlug ? appQuery.data?.at(0) : undefined;

  return { appId: app?.id };
}
