import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { ControlCloud } from "@unkey/ui";
import type { PermissionsFilterField } from "../../filters.schema";
import { useFilters } from "../../hooks/use-filters";

const FIELD_DISPLAY_NAMES: Record<PermissionsFilterField, string> = {
  name: "Name",
  description: "Description",
  roleId: "Role ID",
  roleName: "Role name",
  slug: "Slug",
} as const;

const formatFieldName = (field: string): string => {
  if (field in FIELD_DISPLAY_NAMES) {
    return FIELD_DISPLAY_NAMES[field as PermissionsFilterField];
  }

  return field.charAt(0).toUpperCase() + field.slice(1);
};

export const PermissionsListControlCloud = () => {
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
