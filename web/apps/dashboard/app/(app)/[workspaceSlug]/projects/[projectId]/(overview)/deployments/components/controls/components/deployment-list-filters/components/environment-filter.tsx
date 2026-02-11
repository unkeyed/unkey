import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useFilters } from "../../../../../hooks/use-filters";
import { useProjectData } from "../../../../../../data-provider";

export const EnvironmentFilter = () => {
  const { filters, updateFilters } = useFilters();
  const { environments } = useProjectData();


  return (
    <FilterCheckbox
      options={environments.map((environment, i) => ({
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
