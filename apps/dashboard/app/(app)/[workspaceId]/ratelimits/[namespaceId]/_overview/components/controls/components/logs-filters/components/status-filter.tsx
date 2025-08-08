import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useFilters } from "../../../../../hooks/use-filters";

type StatusOption = {
  id: number;
  status: "passed" | "blocked";
  label: string;
  color: string;
  checked: boolean;
};

const options: StatusOption[] = [
  {
    id: 1,
    status: "passed",
    label: "Passed",
    color: "bg-success-9",
    checked: false,
  },
  {
    id: 2,
    status: "blocked",
    label: "Blocked",
    color: "bg-warning-9",
    checked: false,
  },
];

export const StatusFilter = () => {
  const { filters, updateFilters } = useFilters();
  return (
    <FilterCheckbox
      options={options}
      filterField="status"
      checkPath="status"
      renderOptionContent={(checkbox) => (
        <>
          <div className={`size-2 ${checkbox.color} rounded-[2px]`} />
          <span className="text-accent-12 text-xs">{checkbox.label}</span>
        </>
      )}
      createFilterValue={(option) => ({
        value: option.status,
      })}
      updateFilters={updateFilters}
      filters={filters}
    />
  );
};
