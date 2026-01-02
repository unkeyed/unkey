import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const BucketFilter = ({ buckets }: { buckets: string[] }) => {
  const { filters, updateFilters } = useFilters();

  return (
    <FilterCheckbox
      selectionMode="single"
      showScroll
      options={(buckets ?? []).map((bucket, index) => ({
        label: bucket,
        value: bucket,
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
