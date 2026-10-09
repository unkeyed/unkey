"use client";
import {
  type ProjectApps,
  environments,
  inProjectApps,
} from "@/lib/collections/deploy/environments";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { useLiveQuery } from "@tanstack/react-db";

/** Live environments of every app in the given projects. Idle until `projects` loads. */
export function useProjectEnvironments(projects: ReadonlyArray<ProjectApps> | undefined) {
  const withApps = (projects ?? []).filter((project) => project.apps.length > 0);
  const scopeKey = withApps
    .map((project) => `${project.id}:${project.apps.map((app) => app.id).join(",")}`)
    .join(";");

  const query = useLiveQuery(
    (q) =>
      withApps.length === 0
        ? null
        : q.from({ env: environments }).where(({ env }) => inProjectApps(env, withApps)),
    [scopeKey],
  );

  const { failed } = useCollectionLoad(environments.utils);
  return { ...query, isError: failed };
}
