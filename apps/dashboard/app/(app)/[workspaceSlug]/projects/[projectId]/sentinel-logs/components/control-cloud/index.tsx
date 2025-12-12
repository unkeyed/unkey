import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { ControlCloud } from "@unkey/ui";
import { format } from "date-fns";
import { useSentinelLogsFilters } from "../../hooks/use-sentinel-logs-filters";

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
    case "since":
      return "";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

const formatValue = (value: string | number, field: string): string => {
  if (typeof value === "string" && /^\d+$/.test(value)) {
    const statusFamily = Math.floor(Number.parseInt(value) / 100);
    switch (statusFamily) {
      case 5:
        return "5xx (Error)";
      case 4:
        return "4xx (Warning)";
      case 2:
        return "2xx (Success)";
      default:
        return `${statusFamily}xx`;
    }
  }
  if (typeof value === "number" && (field === "startTime" || field === "endTime")) {
    return format(value, "MMM d, yyyy HH:mm:ss");
  }
  return String(value);
};

export const SentinelLogsControlCloud = () => {
  const { filters, updateFilters, removeFilter } = useSentinelLogsFilters();
  return (
    <ControlCloud
      historicalWindow={HISTORICAL_DATA_WINDOW}
      formatValue={formatValue}
      formatFieldName={formatFieldName}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
};
