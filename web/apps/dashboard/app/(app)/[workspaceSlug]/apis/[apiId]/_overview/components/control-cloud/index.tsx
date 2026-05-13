import { ControlCloud } from "@unkey/ui";
import type { FilterValue } from "@unkey/ui/src/validation/filter.types";
import type { KeysOverviewFilterValue } from "../../filters.schema";
import { useFilters } from "../../hooks/use-filters";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "startTime":
      return "Start time";
    case "endTime":
      return "End time";
    case "outcomes":
      return "Outcome";
    case "names":
      return "Name";
    case "identities":
      return "Identity";
    case "keyIds":
      return "Key ID";
    case "since":
      return "";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

export const KeysOverviewLogsControlCloud = () => {
  const { filters, updateFilters, removeFilter } = useFilters();
  return (
    <ControlCloud
      formatFieldName={formatFieldName}
      filters={filters as FilterValue[]}
      removeFilter={removeFilter}
      updateFilters={(newFilters: FilterValue[]) =>
        updateFilters(newFilters as KeysOverviewFilterValue[])
      }
    />
  );
};
