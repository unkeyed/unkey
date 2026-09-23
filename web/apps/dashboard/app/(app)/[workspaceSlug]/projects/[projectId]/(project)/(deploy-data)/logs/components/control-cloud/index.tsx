"use client";

import { useAppNameById } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useEnvironmentFilterOptions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/environment-filter-options";
import { ControlCloud } from "@unkey/ui";
import { format } from "date-fns";
import { useMemo } from "react";
import { useRuntimeLogsFilters } from "../../hooks/use-runtime-logs-filters";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "startTime":
      return "Start time";
    case "endTime":
      return "End time";
    case "severity":
      return "Severity";
    case "message":
      return "Message";
    case "attributes":
      return "Attributes";
    case "appId":
      return "App";
    case "deploymentId":
      return "Deployment";
    case "environmentId":
      return "Environment";
    case "since":
      return "";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

const formatValue = (
  value: string | number,
  field: string,
  appName?: string,
  environmentName?: string,
): string => {
  if (typeof value === "number" && (field === "startTime" || field === "endTime")) {
    return format(value, "MMM d, yyyy HH:mm:ss");
  }
  if (field === "severity") {
    return value.toString().toUpperCase();
  }
  if (field === "appId") {
    return appName ?? String(value);
  }
  if (field === "environmentId") {
    return environmentName ?? String(value);
  }
  return String(value);
};

export function RuntimeLogsControlCloud() {
  const { filters, removeFilter, updateFilters } = useRuntimeLogsFilters();
  const appNameById = useAppNameById();
  const environmentOptions = useEnvironmentFilterOptions();
  const environmentNameById = useMemo(
    () =>
      new Map(
        environmentOptions.map((environment) => [
          environment.environmentId,
          environment.appName ? `${environment.appName} / ${environment.slug}` : environment.slug,
        ]),
      ),
    [environmentOptions],
  );

  return (
    <ControlCloud
      formatValue={(value, field) =>
        formatValue(
          value,
          field,
          appNameById.get(String(value)),
          environmentNameById.get(String(value)),
        )
      }
      formatFieldName={formatFieldName}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
}
