"use client";

import { DeploymentIdFilter } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/deployment-id-filter";
import { useSentinelLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-sentinel-logs-filters";
import type { LogsFilterValue } from "@/lib/schemas/logs.filter.schema";

export const SentinelDeploymentFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();

  return (
    <DeploymentIdFilter
      filters={filters}
      updateFilters={updateFilters}
      createDeploymentFilter={(value): LogsFilterValue => ({
        id: crypto.randomUUID(),
        field: "deploymentId",
        operator: "is",
        value,
      })}
    />
  );
};
