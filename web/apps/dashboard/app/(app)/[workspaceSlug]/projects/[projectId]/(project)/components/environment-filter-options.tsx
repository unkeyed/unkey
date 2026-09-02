"use client";

import { useMemo } from "react";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useAppNameById } from "./app-filter-options";

// Logs and requests are project-wide and every app has e.g. a "production"
// environment, so the app name is needed to tell same-slug options apart.
export function useEnvironmentFilterOptions(): EnvironmentFilterOption[] {
  const { environments } = useProjectData();
  const appNameById = useAppNameById();

  return useMemo(
    () =>
      environments.map((environment, i) => ({
        id: i,
        slug: environment.slug,
        appName: appNameById.get(environment.appId) ?? null,
        environmentId: environment.id,
        checked: false,
      })),
    [environments, appNameById],
  );
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
