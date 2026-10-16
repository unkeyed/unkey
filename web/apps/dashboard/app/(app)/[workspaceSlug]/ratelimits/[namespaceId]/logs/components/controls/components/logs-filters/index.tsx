import { type FilterItemConfig, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { IconBarsFilterOutline18 } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { RatelimitFilterField } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";
import { IdentifiersFilter } from "./components/identifiers-filter";
import { StatusFilter, statusOptions } from "./components/status-filter";

const FILTER_ITEMS: FilterItemConfig<RatelimitFilterField>[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "e",
    component: <StatusFilter />,
    options: statusOptions.map((status) => ({ value: status.status, label: status.label })),
  },
  {
    id: "identifiers",
    label: "Identifier",
    shortcut: "p",
    component: <IdentifiersFilter />,
  },
];

export const LogsFilters = () => {
  const { filters, updateFilters } = useFilters();
  return (
    <FiltersPopover items={FILTER_ITEMS} activeFilters={filters} updateFilters={updateFilters}>
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
