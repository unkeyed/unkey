"use client";

import {
  renderAppOption,
  useAppFilterOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const RequestAppFilter = () => {
  const { filters, updateFilters } = useRequestLogsFilters();
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
