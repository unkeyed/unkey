import { type FilterItemConfig, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useFilters } from "../../../../hooks/use-filters";
import { IdentifiersFilter } from "./components/identifiers-filter";
import { StatusFilter } from "./components/status-filter";

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

export const LogsFilters = () => {
  const { filters } = useFilters();
  return (
    <FiltersPopover items={FILTER_ITEMS} activeFilters={filters}>
      <div className="group">
        <Button
          size="md"
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 rounded-lg",
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
