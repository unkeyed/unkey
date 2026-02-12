"use client";

import { ControlCloud } from "@unkey/ui";
import { format } from "date-fns";
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

const formatValue = (value: string | number, field: string): string => {
  if (typeof value === "number" && (field === "startTime" || field === "endTime")) {
    return format(value, "MMM d, yyyy HH:mm:ss");
  }
  if (field === "severity") {
    return value.toString().toUpperCase();
  }
  return String(value);
};

export function RuntimeLogsControlCloud() {
  const { filters, removeFilter, updateFilters } = useRuntimeLogsFilters();

  return (
    <ControlCloud
      formatValue={formatValue}
      formatFieldName={formatFieldName}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
}
