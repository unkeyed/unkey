"use client";

import { DeploymentIdFilter } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/deployment-id-filter";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import type { RequestLogsFilterValue } from "@/lib/schemas/request-logs.filter.schema";

export const RequestDeploymentFilter = () => {
  const { filters, updateFilters } = useRequestLogsFilters();

  return (
    <DeploymentIdFilter
      filters={filters}
      updateFilters={updateFilters}
      createDeploymentFilter={(value): RequestLogsFilterValue => ({
        id: crypto.randomUUID(),
        field: "deploymentId",
        operator: "is",
        value,
      })}
    />
  );
};
