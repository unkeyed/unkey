import { useFilters } from "@/app/(app)/[workspaceId]/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { shortenId } from "@/lib/shorten-id";

export const RootKeysFilter = ({
  rootKeys,
}: {
  rootKeys: {
    id: string;
    name: string | null;
  }[];
}) => {
  const { filters, updateFilters } = useFilters();
  return (
    <FilterCheckbox
      showScroll
      options={(rootKeys ?? []).map((rootKey, index) => ({
        label: getRootKeyLabel(rootKey), // Format the root key label when empty and contains no name.
        value: rootKey.id,
        checked: false,
        id: index,
      }))}
      filterField="rootKeys"
      checkPath="value"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.label}</div>
      )}
      createFilterValue={(option) => ({
        value: option.value,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};

// Format the root key label when rootkey contains no name. Splits and formats into 'unkey_1234...5678'
// This is to avoid displaying the full id in the filter.
const getRootKeyLabel = (rootKey: { id: string; name: string | null }) => {
  const id = rootKey.id;
  return rootKey.name ?? shortenId(id);
};
