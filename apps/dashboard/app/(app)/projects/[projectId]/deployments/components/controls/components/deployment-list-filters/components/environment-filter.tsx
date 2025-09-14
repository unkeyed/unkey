import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { useFilters } from "../../../../../hooks/use-filters";

export const EnvironmentFilter = () => {
  const { filters, updateFilters } = useFilters();

  const environments = useLiveQuery((q) => q.from({ environment: collection.environments }));

  return (
    <FilterCheckbox
      options={environments.data.map((environment, i) => ({
        id: i,
        slug: environment.slug,
      }))}
      filterField="environment"
      checkPath="slug"
      selectionMode="single"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs capitalize">{checkbox.slug}</div>
      )}
      createFilterValue={(option) => ({
        value: option.slug,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
