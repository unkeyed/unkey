import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { ControlCloud } from "@/components/logs/control-cloud";
import { useFilters } from "../../hooks/use-filters";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "name":
      return "Name";
    case "slug":
      return "Slug";
    case "description":
      return "Description";
    case "keyName":
      return "Key name";
    case "keyId":
      return "Key ID";
    case "permissionSlug":
      return "Permission slug";
    case "permissionName":
      return "Permission name";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
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
