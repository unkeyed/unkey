"use client";

import {
  renderAppOption,
  useAppFilterOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const RuntimeLogsAppFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const options = useAppFilterOptions();

  return (
    <FilterCheckbox
      options={options}
      filterField="appId"
      checkPath="appId"
      selectionMode="multiple"
      renderOptionContent={renderAppOption}
      createFilterValue={(option) => ({
        value: option.appId,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
