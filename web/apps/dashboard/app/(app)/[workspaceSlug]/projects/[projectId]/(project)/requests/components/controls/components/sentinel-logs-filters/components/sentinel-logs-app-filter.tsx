"use client";

import {
  renderAppOption,
  useAppFilterOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useSentinelLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-sentinel-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const SentinelAppFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();
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
