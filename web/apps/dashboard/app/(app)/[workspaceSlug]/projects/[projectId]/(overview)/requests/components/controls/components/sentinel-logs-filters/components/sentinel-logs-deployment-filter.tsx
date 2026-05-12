"use client";

import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useProjectData } from "../../../../../../data-provider";
import { useSentinelLogsFilters } from "../../../../../hooks/use-sentinel-logs-filters";

export const SentinelDeploymentFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();
  const { deployments } = useProjectData();

  const options = deployments.map((deployment, i) => ({
    id: i,
    slug: deployment.id,
    deploymentId: deployment.id,
    gitBranch: deployment.gitBranch,
    checked: false,
  }));

  return (
    <FilterCheckbox
      className="w-80"
      options={options}
      filterField="deploymentId"
      checkPath="slug"
      showScroll
      maxVisibleOptions={50}
      searchPlaceholder="Search deployments"
      getSearchText={(option) => `${option.gitBranch ?? ""} ${option.slug ?? ""}`}
      selectionMode="multiple"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs flex min-w-0 flex-1 items-center gap-2">
          <span className="font-semibold truncate">{checkbox.gitBranch}</span>
          <span className="font-mono shrink-0 text-accent-11">{checkbox.slug.slice(0, 8)}</span>
        </div>
      )}
      createFilterValue={(option) => ({
        value: option.deploymentId,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
