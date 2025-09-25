import { useFilters } from "@/app/(app)/logs/hooks/use-filters";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { logsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";

export const PathsFilter = () => {
  const { filters, updateFilters } = useFilters();

  const pathOperators = logsFilterFieldConfig.paths.operators;
  const options = pathOperators.map((op) => ({
    id: op,
    label: op,
  }));

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
            operator: id,
            value: text,
          },
        ]);
      }}
    />
  );
};
