import { useFilters } from "@/app/(app)/logs/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import type { GroupedDeploymentStatus } from "../../../../../filters.schema";

type StatusOption = {
  id: number;
  status: GroupedDeploymentStatus;
  display: string;
  label: string;
  color: string;
  checked: boolean;
};

const options: StatusOption[] = [
  {
    id: 1,
    status: "pending",
    display: "Pending",
    label: "Queued",
    color: "bg-gray-9",
    checked: false,
  },
  {
    id: 2,
    status: "building",
    display: "Building",
    label: "In Progress",
    color: "bg-info-9",
    checked: false,
  },
  {
    id: 3,
    status: "completed",
    display: "Active",
    label: "Success",
    color: "bg-success-9",
    checked: false,
  },
  {
    id: 4,
    status: "failed",
    display: "Failed",
    label: "Error",
    color: "bg-error-9",
    checked: false,
  },
];

export const DeploymentStatusFilter = () => {
  const { filters, updateFilters } = useFilters();

  return (
    <FilterCheckbox
      options={options}
      filterField="status"
      checkPath="status"
      renderOptionContent={(checkbox) => (
        <>
          <div className={`size-2 ${checkbox.color} rounded-[2px]`} />
          <span className="text-accent-9 text-xs w-16">{checkbox.display}</span>
          <span className="text-accent-12 text-xs">{checkbox.label}</span>
        </>
      )}
      createFilterValue={(option) => ({
        value: option.status,
        metadata: {
          colorClass: option.color,
        },
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
