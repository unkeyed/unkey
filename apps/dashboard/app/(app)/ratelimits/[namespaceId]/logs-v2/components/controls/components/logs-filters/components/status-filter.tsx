import type { ResponseStatus } from "@/app/(app)/logs-v2/filters.type";
import { FilterCheckbox } from "./filter-checkbox";

type StatusOption = {
  id: number;
  status: ResponseStatus;
  label: string;
  color: string;
  checked: boolean;
};

const options: StatusOption[] = [
  {
    id: 1,
    status: 200,
    label: "Succeeded",
    color: "bg-success-9",
    checked: false,
  },
  {
    id: 2,
    status: 400,
    label: "Rejected",
    color: "bg-error-8",
    checked: false,
  },
];

export const StatusFilter = () => {
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
        metadata: {
          colorClass:
            option.status >= 500
              ? "bg-error-9"
              : option.status >= 400
                ? "bg-warning-8"
                : "bg-success-9",
        },
      })}
    />
  );
};
