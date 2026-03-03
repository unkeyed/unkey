import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { cn } from "@/lib/utils";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";

import {
  type RootKeysFilterField,
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";

const FIELD_DISPLAY_CONFIG: Record<RootKeysFilterField, { label: string; shortcut: string }> = {
  name: { label: "Name", shortcut: "n" },
  start: { label: "Key", shortcut: "k" },
  permission: { label: "Permission", shortcut: "p" },
} as const;

export const RootKeysFilters = () => {
  const { filters, updateFilters } = useFilters();

  // Generate filter items dynamically from schema
  const filterItems = rootKeysListFilterFieldNames.map((fieldName) => {
    const fieldConfig = rootKeysFilterFieldConfig[fieldName];
    const displayConfig = FIELD_DISPLAY_CONFIG[fieldName];

    if (!displayConfig) {
      if (process.env.NODE_ENV !== "production") {
        // Fail-fast in dev to surface schema/display drift
        throw new Error(`Missing display configuration for field: ${fieldName}`);
      }
      // Fail-soft in prod
      return {
        id: fieldName,
        label: fieldName.charAt(0).toUpperCase() + fieldName.slice(1),
        shortcut: "",
        component: null,
      } as const;
    }

    // …rest of mapping logic using displayConfig…
    const options = fieldConfig.operators.map((op) => ({
      id: op,
      label: op,
    }));

    const activeFilter = filters.find((f) => f.field === fieldName);

    return {
      id: fieldName,
      label: displayConfig.label,
      shortcut: displayConfig.shortcut,
      component: (
        <FilterOperatorInput
          label={displayConfig.label}
          options={options}
          defaultOption={activeFilter?.operator}
          defaultText={activeFilter?.value as string}
          onApply={(operator, text) => {
            // Remove existing filters for this field
            const filtersWithoutCurrent = filters.filter((f) => f.field !== fieldName);

            // Add new filter
            updateFilters([
              ...filtersWithoutCurrent,
              {
                field: fieldName,
                id: crypto.randomUUID(),
                operator,
                value: text,
              },
            ]);
          }}
        />
      ),
    };
  });

  return (
    <FiltersPopover items={filterItems} activeFilters={filters}>
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 rounded-lg",
            filters.length > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter root keys"
          aria-haspopup="dialog"
          size="md"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
          {filters.length > 0 && (
            <div className="bg-gray-7 rounded-sm h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {filters.length}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
