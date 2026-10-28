import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/(deploy-data)/requests/hooks/use-request-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { methodOptions } from "../request-logs-filter-items";

export const RequestMethodsFilter = () => {
  const { filters, updateFilters } = useRequestLogsFilters();
  return (
    <FilterCheckbox
      options={methodOptions}
      filterField="methods"
      checkPath="method"
      renderOptionContent={(checkbox) => (
        <div className="text-gray-12 text-xs">{checkbox.method}</div>
      )}
      createFilterValue={(option) => ({
        value: option.method,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
