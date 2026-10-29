import { useFilters } from "@/app/(app)/[workspaceSlug]/logs/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { HttpMethod } from "@unkey/api/models/components";

export const methodOptions = Object.values(HttpMethod).map((method, i) => ({
  id: i + 1,
  method,
  checked: false,
}));

export const MethodsFilter = () => {
  const { filters, updateFilters } = useFilters();
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
