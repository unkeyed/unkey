"use client";

import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useProjectData } from "../../../../../data-provider";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

export const RuntimeLogsEnvironmentFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const { environments } = useProjectData();

  const options = environments.map((environment, i) => ({
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
