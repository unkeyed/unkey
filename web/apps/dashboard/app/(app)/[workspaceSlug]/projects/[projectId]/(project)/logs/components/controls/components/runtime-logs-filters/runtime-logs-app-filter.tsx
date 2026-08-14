"use client";

import { AppEnvironmentFilter } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-environment-filter";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import type { RuntimeLogsFilterValue } from "@/lib/schemas/runtime-logs.filter.schema";

const createFilter = (field: "appId" | "environmentId", value: string): RuntimeLogsFilterValue => ({
  id: crypto.randomUUID(),
  field,
  operator: "is",
  value,
});

export const RuntimeLogsAppFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();

  return (
    <AppEnvironmentFilter
      filters={filters}
      updateFilters={updateFilters}
      createFilter={createFilter}
    />
  );
};
