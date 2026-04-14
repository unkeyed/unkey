import { ControlCloud } from "@unkey/ui";
import { useFilters } from "../hooks/use-filters";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "startTime":
      return "Start time";
    case "endTime":
      return "End time";
    case "since":
      return "";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

export const ApiListControlCloud = () => {
  const { filters, updateFilters, removeFilter } = useFilters();
  return (
    <ControlCloud
      formatFieldName={formatFieldName}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
};
