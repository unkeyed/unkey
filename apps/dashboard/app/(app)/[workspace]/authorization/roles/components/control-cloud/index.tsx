import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { ControlCloud } from "@unkey/ui";
import type { RolesFilterField } from "../../filters.schema";
import { useFilters } from "../../hooks/use-filters";

const FIELD_DISPLAY_NAMES: Record<RolesFilterField, string> = {
  name: "Name",
  description: "Description",
  permissionSlug: "Permission slug",
  permissionName: "Permission name",
  keyId: "Key ID",
  keyName: "Key name",
} as const;

const formatFieldName = (field: string): string => {
  if (field in FIELD_DISPLAY_NAMES) {
    return FIELD_DISPLAY_NAMES[field as RolesFilterField];
  }

  return field.charAt(0).toUpperCase() + field.slice(1);
};

export const RolesListControlCloud = () => {
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
