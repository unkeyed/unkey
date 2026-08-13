"use client";

import { DeploymentIdFilter } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/deployment-id-filter";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import type { LogsFilterValue } from "@/lib/schemas/logs.filter.schema";

export const RequestDeploymentFilter = () => {
  const { filters, updateFilters } = useRequestLogsFilters();

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
