"use client";

import { DeploymentIdFilter } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/deployment-id-filter";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import type { RuntimeLogsFilterValue } from "@/lib/schemas/runtime-logs.filter.schema";

export const RuntimeLogsDeploymentFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();

  return (
    <DeploymentIdFilter
      filters={filters}
      updateFilters={updateFilters}
      createDeploymentFilter={(value): RuntimeLogsFilterValue => ({
        id: crypto.randomUUID(),
        field: "deploymentId",
        operator: "is",
        value,
      })}
    />
  );
};
