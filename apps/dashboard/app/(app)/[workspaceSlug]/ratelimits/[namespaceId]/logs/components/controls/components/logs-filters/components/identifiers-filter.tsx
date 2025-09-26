import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { ratelimitFilterFieldConfig } from "../../../../../filters.schema";
import { useFilters } from "../../../../../hooks/use-filters";

export const IdentifiersFilter = () => {
  const { filters, updateFilters } = useFilters();

  const identifierOperators = ratelimitFilterFieldConfig.identifiers.operators;
  const options = identifierOperators.map((op) => ({
    id: op,
    label: op,
  }));

  const activeIdentifierFilter = filters.find((f) => f.field === "identifiers");

  return (
    <FilterOperatorInput
      label="Identifier"
      options={options}
      defaultOption={activeIdentifierFilter?.operator}
      defaultText={activeIdentifierFilter?.value as string}
      onApply={(id, text) => {
        const activeFiltersWithoutIdentifiers = filters.filter((f) => f.field !== "identifiers");
        updateFilters([
          ...activeFiltersWithoutIdentifiers,
          {
            field: "identifiers",
            id: crypto.randomUUID(),
            operator: id,
            value: text,
          },
        ]);
      }}
    />
  );
};
