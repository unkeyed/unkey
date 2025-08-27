import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { ControlCloud } from "@unkey/ui";
import type { RootKeysFilterField } from "../../filters.schema";
import { useFilters } from "../../hooks/use-filters";

const FIELD_DISPLAY_NAMES: Record<RootKeysFilterField, string> = {
  name: "Name",
  start: "Key",
  permission: "Permission",
} as const;

const formatFieldName = (field: string): string => {
  if (field in FIELD_DISPLAY_NAMES) {
    return FIELD_DISPLAY_NAMES[field as RootKeysFilterField];
  }

  return field.charAt(0).toUpperCase() + field.slice(1);
};

export const RootKeysListControlCloud = () => {
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
