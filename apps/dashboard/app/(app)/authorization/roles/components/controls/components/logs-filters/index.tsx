import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";

import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { rolesFilterFieldConfig } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsFilters = () => {
  const { filters, updateFilters } = useFilters();

  const options = rolesFilterFieldConfig.name.operators.map((op) => ({
    id: op,
    label: op,
  }));
  const activeNameFilter = filters.find((f) => f.field === "name");
  const activeSlugFilter = filters.find((f) => f.field === "slug");
  const activeDescriptionFilter = filters.find((f) => f.field === "description");

  return (
    <FiltersPopover
      items={[
        {
          id: "names",
          label: "Name",
          shortcut: "n",
          component: (
            <FilterOperatorInput
              label="Name"
              options={options}
              defaultOption={activeNameFilter?.operator}
              defaultText={activeNameFilter?.value as string}
              onApply={(id, text) => {
                const activeFiltersWithoutNames = filters.filter((f) => f.field !== "name");
                updateFilters([
                  ...activeFiltersWithoutNames,
                  {
                    field: "name",
                    id: crypto.randomUUID(),
                    operator: id,
                    value: text,
                  },
                ]);
              }}
            />
          ),
        },
        {
          id: "slug",
          label: "Slug",
          shortcut: "s",
          component: (
            <FilterOperatorInput
              label="Slug"
              options={options}
              defaultOption={activeSlugFilter?.operator}
              defaultText={activeSlugFilter?.value as string}
              onApply={(id, text) => {
                const activeFiltersWithoutNames = filters.filter((f) => f.field !== "slug");
                updateFilters([
                  ...activeFiltersWithoutNames,
                  {
                    field: "slug",
                    id: crypto.randomUUID(),
                    operator: id,
                    value: text,
                  },
                ]);
              }}
            />
          ),
        },
        {
          id: "description",
          label: "Description",
          shortcut: "d",
          component: (
            <FilterOperatorInput
              label="Description"
              options={options}
              defaultOption={activeDescriptionFilter?.operator}
              defaultText={activeDescriptionFilter?.value as string}
              onApply={(id, text) => {
                const activeFiltersWithoutDescriptions = filters.filter(
                  (f) => f.field !== "description",
                );
                updateFilters([
                  ...activeFiltersWithoutDescriptions,
                  {
                    field: "description",
                    id: crypto.randomUUID(),
                    operator: id,
                    value: text,
                  },
                ]);
              }}
            />
          ),
        },
      ]}
      activeFilters={filters}
    >
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
