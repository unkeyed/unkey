import { ControlCloud } from "@unkey/ui";
import { format } from "date-fns";
import { useAppNameById } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useRequestLogsFilters } from "../../hooks/use-request-logs-filters";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "startTime":
      return "Start time";
    case "endTime":
      return "End time";
    case "status":
      return "Status";
    case "paths":
      return "Path";
    case "methods":
      return "Method";
    case "appId":
      return "App";
    case "deploymentId":
      return "Deployment ID";
    case "since":
      return "";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

const RANGE_STATUS_VALUES = new Set(["200", "300", "400", "500"]);

const formatStatusCode = (raw: string): string => {
  const code = Number.parseInt(raw);
  if (RANGE_STATUS_VALUES.has(raw)) {
    if (code >= 500) {
      return "5xx (Error)";
    }
    if (code >= 400) {
      return "4xx (Warning)";
    }
    if (code >= 300) {
      return "3xx (Redirect)";
    }
    return "2xx (Success)";
  }
  if (code >= 500) {
    return `${raw} (Error)`;
  }
  if (code >= 400) {
    return `${raw} (Warning)`;
  }
  if (code >= 300) {
    return `${raw} (Redirect)`;
  }
  return `${raw} (Success)`;
};

const formatValue = (value: string | number, field: string, appName?: string): string => {
  if (field === "status" && /^\d+$/.test(String(value))) {
    return formatStatusCode(String(value));
  }
  if (typeof value === "number" && (field === "startTime" || field === "endTime")) {
    return format(value, "MMM d, yyyy HH:mm:ss");
  }
  if (field === "appId") {
    return appName ?? String(value);
  }
  return String(value);
};

export const RequestLogsControlCloud = () => {
  const { filters, updateFilters, removeFilter } = useRequestLogsFilters();
  const appNameById = useAppNameById();
  return (
    <ControlCloud
      formatValue={(value, field) => formatValue(value, field, appNameById.get(String(value)))}
      formatFieldName={formatFieldName}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
};
