import { logsFilterFieldConfig } from "@/app/(app)/logs/filters.schema";
import { useFilters } from "@/app/(app)/logs/hooks/use-filters";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { useState } from "react";

export const PathsFilter = () => {
  const { filters, updateFilters } = useFilters();
  const activeFilter = filters.find((f) => f.field === "paths");
  const [searchText, setSearchText] = useState(activeFilter?.value.toString() ?? "");
  const [isFocused, setIsFocused] = useState(false);

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
