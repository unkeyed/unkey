"use client";

import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { cn } from "@/lib/utils";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";

import {
  identitiesFilterFieldConfig,
  identitiesListFilterFieldNames,
} from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";

type PopoverField = (typeof identitiesListFilterFieldNames)[number];

const FIELD_DISPLAY_CONFIG: Record<PopoverField, { label: string; shortcut: string }> = {
  externalId: { label: "External ID", shortcut: "e" },
} as const;

export const IdentitiesFilters = () => {
  const { filters, updateFilters } = useFilters();

  const filterItems = identitiesListFilterFieldNames.map((fieldName) => {
    const fieldConfig = identitiesFilterFieldConfig[fieldName];
    const displayConfig = FIELD_DISPLAY_CONFIG[fieldName];

    if (!displayConfig) {
      if (process.env.NODE_ENV !== "production") {
        throw new Error(`Missing display configuration for field: ${fieldName}`);
      }
      return {
        id: fieldName,
        label: fieldName.charAt(0).toUpperCase() + fieldName.slice(1),
        shortcut: "",
        component: null,
      } as const;
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
            const filtersWithoutCurrent = filters.filter((f) => f.field !== fieldName);
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
          aria-label="Filter identities"
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
