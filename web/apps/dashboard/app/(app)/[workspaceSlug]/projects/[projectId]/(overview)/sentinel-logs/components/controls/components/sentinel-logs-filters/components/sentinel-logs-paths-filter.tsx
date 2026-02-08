import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { logsFilterFieldConfig as sentinelLogsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";
import { useSentinelLogsFilters } from "../../../../../hooks/use-sentinel-logs-filters";

export const SentinelPathsFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();

  const options = [{ id: "contains" as const, label: "contains" }];

  const activePathFilter = filters.find((f) => f.field === "paths");

  return (
    <FilterOperatorInput
      label="Path"
      options={options}
      defaultOption={activePathFilter?.operator}
      defaultText={activePathFilter?.value as string}
      onApply={(id, text) => {
        const activeFiltersWithoutPaths = filters.filter((f) => f.field !== "paths");
        updateFilters([
          ...activeFiltersWithoutPaths,
          {
            field: "paths",
            id: crypto.randomUUID(),
            operator: "contains",
            value: text,
          },
        ]);
      }}
    />
  );
};
