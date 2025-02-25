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
  return (
    <FilterCheckbox
      showScroll
      options={(users ?? []).map((user, index) => ({
        ...user,
        checked: false,
        id: index,
      }))}
      filterField="status"
      checkPath="status"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.label}</div>
      )}
      createFilterValue={(option) => ({
        value: option.value,
      })}
    />
  );
};
