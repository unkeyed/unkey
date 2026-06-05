"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const RuntimeLogsDeploymentFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
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
        <div className="text-accent-12 text-xs flex items-center gap-4.5">
          <span className="text-accent-9">{checkbox.gitBranch}</span>
          <span className="font-mono">{checkbox.slug}</span>
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
