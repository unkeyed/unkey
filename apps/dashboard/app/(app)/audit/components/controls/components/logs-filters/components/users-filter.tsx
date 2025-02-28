import { useFilters } from "@/app/(app)/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const UsersFilter = ({
  users,
}: {
  users: {
    name: string;
    id: string;
  }[];
}) => {
  const { filters, updateFilters } = useFilters();
  return (
    <FilterCheckbox
      showScroll
      options={users.map((user, index) => ({
        ...user,
        checked: false,
        id: index,
      }))}
      filterField="users"
      checkPath="id"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.name}</div>
      )}
      createFilterValue={(option) => ({
        value: option.id,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
