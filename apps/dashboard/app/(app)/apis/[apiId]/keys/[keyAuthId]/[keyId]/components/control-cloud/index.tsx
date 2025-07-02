import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { ControlCloud } from "@unkey/ui";
import { useFilters } from "../../hooks/use-filters";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "startTime":
      return "Start time";
    case "endTime":
      return "End time";
    case "outcomes":
      return "Outcome";
    case "since":
      return "";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

export const KeysDetailsLogsControlCloud = () => {
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
