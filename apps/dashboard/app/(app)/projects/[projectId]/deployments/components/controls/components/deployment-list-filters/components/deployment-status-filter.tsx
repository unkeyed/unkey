import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import type { GroupedDeploymentStatus } from "../../../../../filters.schema";
import { deploymentListFilterFieldConfig } from "../../../../../filters.schema";
import { useFilters } from "../../../../../hooks/use-filters";

type StatusOption = {
  id: number;
  status: GroupedDeploymentStatus;
  display: string;
  label: string;
  checked: boolean;
};

const baseOptions: StatusOption[] = [
  {
    id: 1,
    status: "pending",
    display: "Pending",
    label: "Queued",
    checked: false,
  },
  {
    id: 2,
    status: "building",
    display: "Building",
    label: "In Progress",
    checked: false,
  },
  {
    id: 3,
    status: "completed",
    display: "Active",
    label: "Success",
    checked: false,
  },
  {
    id: 4,
    status: "failed",
    display: "Failed",
    label: "Error",
    checked: false,
  },
];

export const DeploymentStatusFilter = () => {
  const { filters, updateFilters } = useFilters();
  const getColorClass = deploymentListFilterFieldConfig.status.getColorClass;

  return (
    <FilterCheckbox
      options={baseOptions}
      filterField="status"
      checkPath="status"
      renderOptionContent={(checkbox) => (
        <>
          <div className={`size-2 ${getColorClass?.(checkbox.status)} rounded-[2px]`} />
          <span className="text-accent-9 text-xs w-16">{checkbox.display}</span>
          <span className="text-accent-12 text-xs">{checkbox.label}</span>
        </>
      )}
      createFilterValue={(option) => ({
        value: option.status,
        metadata: {
          colorClass: getColorClass?.(option.status),
        },
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
