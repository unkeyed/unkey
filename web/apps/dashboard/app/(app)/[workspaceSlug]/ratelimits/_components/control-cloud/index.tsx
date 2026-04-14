import { ControlCloud } from "@unkey/ui";
import { useNamespaceListFilters } from "../hooks/use-namespace-list-filters";

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

export const NamespaceListControlCloud = () => {
  const { filters, updateFilters, removeFilter } = useNamespaceListFilters();
  return (
    <ControlCloud
      formatFieldName={formatFieldName}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
};
