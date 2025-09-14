import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import type { GroupedDeploymentStatus } from "../../../../../filters.schema";
import { deploymentListFilterFieldConfig } from "../../../../../filters.schema";
import { useFilters } from "../../../../../hooks/use-filters";

type StatusOption = {
  id: number;
  status: GroupedDeploymentStatus;
  display: string;
};

const baseOptions: StatusOption[] = [
  {
    id: 1,
    status: "pending",
    display: "Pending",
  },
  {
    id: 2,
    status: "deploying",
    display: "Deploying",
  },
  {
    id: 3,
    status: "ready",
    display: "Ready",
  },
  {
    id: 4,
    status: "failed",
    display: "Failed",
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
