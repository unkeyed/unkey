import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { ControlCloud } from "@unkey/ui";
import type { DeploymentListFilterField } from "../../filters.schema";
import { useFilters } from "../../hooks/use-filters";

const FIELD_DISPLAY_NAMES: Record<DeploymentListFilterField, string> = {
  status: "Status",
  environment: "Environment",
  branch: "Branch",
  startTime: "Start Time",
  endTime: "End Time",
  since: "Since",
} as const;

const formatFieldName = (field: string): string => {
  if (field in FIELD_DISPLAY_NAMES) {
    return FIELD_DISPLAY_NAMES[field as DeploymentListFilterField];
  }
  // Fallback for any missing fields
  return field
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (str) => str.toUpperCase())
    .trim();
};

export const DeploymentsListControlCloud = () => {
  const { filters, updateFilters, removeFilter } = useFilters();

  return (
    <ControlCloud
      historicalWindow={HISTORICAL_DATA_WINDOW}
      formatFieldName={formatFieldName}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
};
