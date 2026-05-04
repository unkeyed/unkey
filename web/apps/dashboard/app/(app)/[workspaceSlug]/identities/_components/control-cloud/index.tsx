"use client";

import { ControlCloud } from "@unkey/ui";
import { format } from "date-fns";
import type { IdentitiesFilterField } from "../../filters.schema";
import { useFilters } from "../../hooks/use-filters";

const FIELD_DISPLAY_NAMES: Record<IdentitiesFilterField, string> = {
  externalId: "External ID",
  lastUsedStart: "Last Used From",
  lastUsedEnd: "Last Used To",
  lastUsedSince: "Last Used",
};

const formatFieldName = (field: string): string => {
  return FIELD_DISPLAY_NAMES[field as IdentitiesFilterField] ?? field;
};

const formatValue = (value: string | number, field: string): string => {
  if (typeof value === "number" && (field === "lastUsedStart" || field === "lastUsedEnd")) {
    return format(value, "MMM d, yyyy HH:mm");
  }
  return String(value);
};

export const IdentitiesListControlCloud = () => {
  const { filters, updateFilters, removeFilter } = useFilters();

  return (
    <ControlCloud
      formatFieldName={formatFieldName}
      formatValue={formatValue}
      filters={filters}
      removeFilter={removeFilter}
      updateFilters={updateFilters}
    />
  );
};
