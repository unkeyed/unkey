"use client";

import { AppEnvironmentFilter } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-environment-filter";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import type { RequestLogsFilterValue } from "@/lib/schemas/request-logs.filter.schema";

const createFilter = (field: "appId" | "environmentId", value: string): RequestLogsFilterValue => ({
  id: crypto.randomUUID(),
  field,
  operator: "is",
  value,
});

export const RequestAppFilter = () => {
  const { filters, updateFilters } = useRequestLogsFilters();

  return (
    <AppEnvironmentFilter
      filters={filters}
      updateFilters={updateFilters}
      createFilter={createFilter}
    />
  );
};
