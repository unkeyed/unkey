"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/apps/[appSlug]/(overview)/data-provider";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";

// Logs and requests are project-wide and every app has e.g. a "production"
// environment, so the app name is needed to tell same-slug options apart.
export function useEnvironmentFilterOptions(): EnvironmentFilterOption[] {
  const { environments, projectId } = useProjectData();

  const apps = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const appNameById = new Map((apps.data ?? []).map((app) => [app.id, app.name]));

  return environments.map((environment, i) => ({
    id: i,
    slug: environment.slug,
    appName: appNameById.get(environment.appId) ?? null,
    environmentId: environment.id,
    checked: false,
  }));
}

export function renderEnvironmentOption(option: EnvironmentFilterOption) {
  return (
    <div className="text-accent-12 text-xs">
      <span className="capitalize">{option.slug}</span>
      {option.appName ? <span className="text-accent-9"> · {option.appName}</span> : null}
    </div>
  );
}

type EnvironmentFilterOption = {
  id: number;
  slug: string;
  appName: string | null;
  environmentId: string;
  checked: boolean;
};
