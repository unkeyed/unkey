"use client";

import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useProjectData } from "../../../../../../data-provider";
import { useSentinelLogsFilters } from "../../../../../hooks/use-sentinel-logs-filters";

export const SentinelEnvironmentFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();
  const { environments } = useProjectData()

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
