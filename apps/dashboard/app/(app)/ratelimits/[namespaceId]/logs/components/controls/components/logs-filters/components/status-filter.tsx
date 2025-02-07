import { FilterCheckbox } from "./filter-checkbox";

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
    />
  );
};
