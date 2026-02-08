import { useFilters } from "@/app/(app)/[workspaceSlug]/logs/hooks/use-filters";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { logsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";

export const PathsFilter = () => {
  const { filters, updateFilters } = useFilters();

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
