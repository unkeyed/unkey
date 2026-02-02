"use client";

import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

type SeverityOption = {
  id: number;
  severity: string;
  label: string;
  color: string;
  checked: boolean;
};

const options: SeverityOption[] = [
  {
    id: 1,
    severity: "ERROR",
    label: "Error",
    color: "bg-error-9",
    checked: false,
  },
  {
    id: 2,
    severity: "WARN",
    label: "Warning",
    color: "bg-warning-8",
    checked: false,
  },
  {
    id: 3,
    severity: "INFO",
    label: "Info",
    color: "bg-info-9",
    checked: false,
  },
  {
    id: 4,
    severity: "DEBUG",
    label: "Debug",
    color: "bg-grayA-9",
    checked: false,
  },
];

export function RuntimeLogsSeverityFilter() {
  const { filters, updateFiltersFromArray } = useRuntimeLogsFilters();

  return (
    <FilterCheckbox
      options={options}
      filterField="severity"
      checkPath="severity"
      renderOptionContent={(checkbox) => (
        <>
          <div className={`size-2 ${checkbox.color} rounded-[2px]`} />
          <span className="text-accent-12 text-xs">{checkbox.label}</span>
        </>
      )}
      createFilterValue={(option) => ({
        value: option.severity,
        metadata: {
          colorClass: option.color,
          label: option.label,
        },
      })}
      filters={filters}
      updateFilters={updateFiltersFromArray}
    />
  );
}
