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
      options={options}
      filterField="deploymentId"
      checkPath="slug"
      selectionMode="multiple"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">
          <span className="font-medium">{checkbox.gitBranch}</span>
          <span className="text-accent-9 ml-1 font-mono">{checkbox.slug.slice(0, 8)}</span>
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
