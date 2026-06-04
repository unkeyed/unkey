"use client";

import {
  renderEnvironmentOption,
  useEnvironmentFilterOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/(project)/components/environment-filter-options";
import { useSentinelLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/(project)/requests/hooks/use-sentinel-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const SentinelEnvironmentFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();
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
