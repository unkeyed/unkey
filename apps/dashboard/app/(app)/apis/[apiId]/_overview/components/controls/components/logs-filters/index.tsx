import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";

import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
// import { keysOverviewFilterFieldConfig } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsFilters = () => {
  const { filters } = useFilters();

  // const identifierOperators =
  //   keysOverviewFilterFieldConfig..operators;
  // const options = identifierOperators.map((op) => ({
  //   id: op,
  //   label: op,
  // }));

  // const activeIdentifierFilter = filters.find((f) => f.field === "identifiers");
  return (
    <FiltersPopover
      items={[
        {
          id: "identifiers",
          label: "Identifier",
          shortcut: "p",
          component: (
            <FilterOperatorInput
              label="Identifier"
              options={[]}
              // defaultOption={activeIdentifierFilter?.operator}
              // defaultText={activeIdentifierFilter?.value as string}
              onApply={() => {
                return;
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
            "group-data-[state=open]:bg-gray-4 px-2",
            filters.length > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
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
