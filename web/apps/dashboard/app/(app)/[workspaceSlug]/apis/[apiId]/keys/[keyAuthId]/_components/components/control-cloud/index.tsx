import { ControlCloud } from "@unkey/ui";
import { useFilters } from "../../hooks/use-filters";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "names":
      return "Name";
    case "identities":
      return "Identity";
    case "keyIds":
      return "Key ID";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

export const KeysListControlCloud = () => {
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
