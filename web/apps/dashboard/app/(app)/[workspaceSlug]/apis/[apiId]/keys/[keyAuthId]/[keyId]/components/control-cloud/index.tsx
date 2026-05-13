import { ControlCloud } from "@unkey/ui";
import type { FilterValue } from "@unkey/ui/src/validation/filter.types";
import type { KeyDetailsFilterValue } from "../../filters.schema";
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
      formatFieldName={formatFieldName}
      filters={filters as FilterValue[]}
      removeFilter={removeFilter}
      updateFilters={(newFilters: FilterValue[]) =>
        updateFilters(newFilters as KeyDetailsFilterValue[])
      }
    />
  );
};
