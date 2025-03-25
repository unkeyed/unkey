import { useFilters } from "@/app/(app)/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

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
        label: getRootKeyLabel(rootKey),
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

const getRootKeyLabel = (rootKey: { id: string; name: string | null }) => {
  const id = rootKey.id;
  const prefix = id.substring(0, id.indexOf("_") + 1);
  const obfuscatedMiddle =
    id.substring(id.indexOf("_") + 1, id.indexOf("_") + 5).length > 0 ? "..." : "";
  const nextFour = id.substring(id.indexOf("_") + 1, id.indexOf("_") + 5);
  const lastFour = id.substring(id.length - 4);
  return rootKey.name ?? prefix + nextFour + obfuscatedMiddle + lastFour;
};
