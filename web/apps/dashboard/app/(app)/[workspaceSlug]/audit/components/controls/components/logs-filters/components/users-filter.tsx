import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const UsersFilter = ({
  users,
}: {
  users:
    | {
        label: string;
        value: string;
      }[]
    | null;
}) => {
  const { filters, updateFilters } = useFilters();
  return (
    <FilterCheckbox
      showScroll
      options={(users ?? []).map((user, index) => ({
        ...user,
        checked: false,
        id: index,
      }))}
      filterField="users"
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
