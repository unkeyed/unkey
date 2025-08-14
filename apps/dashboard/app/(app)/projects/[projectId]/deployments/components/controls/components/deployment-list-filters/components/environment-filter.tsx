import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useFilters } from "../../../../../hooks/use-filters";

type EnvironmentOption = {
  id: number;
  environment: string;
  checked: boolean;
};

const options: EnvironmentOption[] = [
  { id: 1, environment: "production", checked: false },
  { id: 2, environment: "preview", checked: false },
] as const;

export const EnvironmentFilter = () => {
  const { filters, updateFilters } = useFilters();

  return (
    <FilterCheckbox
      options={options}
      filterField="environment"
      checkPath="environment"
      selectionMode="single"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs capitalize">{checkbox.environment}</div>
      )}
      createFilterValue={(option) => ({
        value: option.environment,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
