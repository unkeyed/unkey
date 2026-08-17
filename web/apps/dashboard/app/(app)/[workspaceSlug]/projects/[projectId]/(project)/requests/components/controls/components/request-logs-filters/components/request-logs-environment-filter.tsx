"use client";

import {
  renderEnvironmentOption,
  useEnvironmentFilterOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/environment-filter-options";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const RequestEnvironmentFilter = () => {
  const { filters, updateFilters } = useRequestLogsFilters();
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
