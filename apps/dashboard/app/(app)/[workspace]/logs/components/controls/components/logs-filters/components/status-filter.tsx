import { useFilters } from "@/app/(app)/[workspace]/logs/hooks/use-filters";
import type { ResponseStatus } from "@/app/(app)/[workspace]/logs/types";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

type StatusOption = {
  id: number;
  status: ResponseStatus;
  display: string;
  label: string;
  color: string;
  checked: boolean;
};

const options: StatusOption[] = [
  {
    id: 1,
    status: 200,
    display: "2xx",
    label: "Success",
    color: "bg-success-9",
    checked: false,
  },
  {
    id: 2,
    status: 400,
    display: "4xx",
    label: "Warning",
    color: "bg-warning-8",
    checked: false,
  },
  {
    id: 3,
    status: 500,
    display: "5xx",
    label: "Error",
    color: "bg-error-9",
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
          <span className="text-accent-9 text-xs">{checkbox.display}</span>
          <span className="text-accent-12 text-xs">{checkbox.label}</span>
        </>
      )}
      createFilterValue={(option) => ({
        value: option.status,
        metadata: {
          colorClass:
            option.status >= 500
              ? "bg-error-9"
              : option.status >= 400
                ? "bg-warning-8"
                : "bg-success-9",
        },
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
