"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

export const RuntimeLogsEnvironmentFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const { environments, projectId } = useProjectData();

  // Logs are project-wide and every app has e.g. a "production" environment,
  // so the app name is needed to tell same-slug options apart.
  const apps = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const appNameById = new Map((apps.data ?? []).map((app) => [app.id, app.name]));

  const options = environments.map((environment, i) => ({
    id: i,
    slug: environment.slug,
    appName: appNameById.get(environment.appId) ?? null,
    environmentId: environment.id,
    checked: false,
  }));

  return (
    <FilterCheckbox
      options={options}
      filterField="environmentId"
      checkPath="environmentId"
      selectionMode="multiple"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">
          <span className="capitalize">{checkbox.slug}</span>
          {checkbox.appName ? <span className="text-accent-9"> · {checkbox.appName}</span> : null}
        </div>
      )}
      createFilterValue={(option) => ({
        value: option.environmentId,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
