import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";

export const RootKeysFilter = ({
  rootKeys,
}: {
  rootKeys: {
    id: string;
    name: string | null;
  }[];
}) => {
  return (
    <FilterCheckbox
      showScroll
      options={(rootKeys ?? []).map((rootKey, index) => ({
        label: rootKey.name,
        value: rootKey.id,
        checked: false,
        id: index,
      }))}
      filterField="status"
      checkPath="status"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.label}</div>
      )}
      createFilterValue={(option) => ({
        value: option.value,
      })}
    />
  );
};
