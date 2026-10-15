import {
  type FilterItemConfig,
  FiltersPopover,
  createFilterOptionBuilder,
} from "@/components/logs/checkbox/filters-popover";
import { IconBarsFilterOutline18 } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo } from "react";
import type { RatelimitFilterField, RatelimitFilterValue } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";
import { IdentifiersFilter } from "./components/identifiers-filter";
import { StatusFilter, statusOptions } from "./components/status-filter";

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "e",
    component: <StatusFilter />,
  },
  {
    id: "identifiers",
    label: "Identifier",
    shortcut: "p",
    component: <IdentifiersFilter />,
  },
];

const createFilter = (field: RatelimitFilterField, value: string): RatelimitFilterValue => ({
  id: crypto.randomUUID(),
  field,
  operator: "is",
  value,
});

export const LogsFilters = () => {
  const { filters, updateFilters } = useFilters();
  const searchItems = useMemo(() => {
    const option = createFilterOptionBuilder({ filters, updateFilters, createFilter });
    return statusOptions.map((status) =>
      option({ field: "status", value: status.status, label: status.label, path: ["Status"] }),
    );
  }, [filters, updateFilters]);
  return (
    <FiltersPopover items={FILTER_ITEMS} searchItems={searchItems} activeFilters={filters}>
      <div className="group">
        <Button
          size="md"
          variant="ghost"
          className={cn(
            "group-data-popup-open:bg-gray-4 px-2 rounded-lg",
            filters.length > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <IconBarsFilterOutline18 className="text-gray-9 size-4" />
          <span className="text-gray-12 font-medium text-sm">Filter</span>
          {filters.length > 0 && <Badge variant="count">{filters.length}</Badge>}
        </Button>
      </div>
    </FiltersPopover>
  );
};
