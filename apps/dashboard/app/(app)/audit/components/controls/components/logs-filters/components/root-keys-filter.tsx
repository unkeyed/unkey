import { useFilters } from "@/app/(app)/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const RootKeysFilter = ({
  rootKeys,
}: {
  rootKeys: {
    id: string;
    name: string | null;
  }[];
}) => {
  const { filters, updateFilters } = useFilters();
  return (
    <FilterCheckbox
      showScroll
      options={(rootKeys ?? []).map((rootKey, index) => ({
        label: rootKey.name ?? rootKey.id.slice(0, 10),
        value: rootKey.id,
        checked: false,
        id: index,
      }))}
      filterField="rootKeys"
      checkPath="value"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.label}</div>
      )}
      createFilterValue={(option) => ({
        value: option.value,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
