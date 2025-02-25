import { useFilters } from "@/app/(app)/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const BucketFilter = ({
  bucketFilter,
}: {
  bucketFilter: {
    id: string;
    name: string | null;
  }[];
}) => {
  const { filters, updateFilters } = useFilters();

  return (
    <FilterCheckbox
      selectionMode="single"
      showScroll
      options={(bucketFilter ?? []).map((rootKey, index) => ({
        label: rootKey.name,
        value: rootKey.name,
        checked: false,
        id: index,
      }))}
      filterField="bucket"
      checkPath="value"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.label}</div>
      )}
      createFilterValue={(option) => ({
        value: option.value ?? "",
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
