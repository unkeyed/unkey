"use client";

import {
  renderEnvironmentOption,
  useEnvironmentFilterOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/(project)/components/environment-filter-options";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/(project)/logs/hooks/use-runtime-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const RuntimeLogsEnvironmentFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const options = useEnvironmentFilterOptions();

  return (
    <FilterCheckbox
      options={options}
      filterField="environmentId"
      checkPath="environmentId"
      selectionMode="multiple"
      renderOptionContent={renderEnvironmentOption}
      createFilterValue={(option) => ({
        value: option.environmentId,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
