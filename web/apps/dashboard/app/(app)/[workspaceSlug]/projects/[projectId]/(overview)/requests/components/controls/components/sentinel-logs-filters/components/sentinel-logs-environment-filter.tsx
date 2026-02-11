"use client";

import { useProject } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/layout-provider";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useSentinelLogsFilters } from "../../../../../hooks/use-sentinel-logs-filters";

export const SentinelEnvironmentFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();
  const { projectId } = useProject();

  const environments = useLiveQuery(
    (q) =>
      q
        .from({ environment: collection.environments })
        .where(({ environment }) => eq(environment.projectId, projectId)),
    [projectId],
  );

  const options = environments.data.map((environment, i) => ({
    id: i,
    slug: environment.slug,
    environmentId: environment.id,
    checked: false,
  }));

  return (
    <FilterCheckbox
      options={options}
      filterField="environmentId"
      checkPath="slug"
      selectionMode="multiple"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs capitalize">{checkbox.slug}</div>
      )}
      createFilterValue={(option) => ({
        value: option.environmentId,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
