import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  type PermissionsFilterField,
  permissionsFilterFieldConfig,
  permissionsListFilterFieldNames,
} from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";

const FIELD_DISPLAY_CONFIG: Record<PermissionsFilterField, { label: string; shortcut: string }> = {
  name: { label: "Name", shortcut: "n" },
  description: { label: "Description", shortcut: "d" },
  slug: { label: "Slug", shortcut: "k" },
  roleName: { label: "Role name", shortcut: "p" },
  roleId: { label: "Role ID", shortcut: "m" },
} as const;

export const LogsFilters = () => {
  const { filters, updateFilters } = useFilters();

  // Generate filter items dynamically from schema
  const filterItems = permissionsListFilterFieldNames.map((fieldName) => {
    const fieldConfig = permissionsFilterFieldConfig[fieldName];
    const displayConfig = FIELD_DISPLAY_CONFIG[fieldName];

    if (!displayConfig) {
      throw new Error(`Missing display configuration for field: ${fieldName}`);
    }

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
          aria-label="Filter logs"
          aria-haspopup="true"
          size="md"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
          {filters.length > 0 && (
            <div className="bg-gray-7 rounded h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {filters.length}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
