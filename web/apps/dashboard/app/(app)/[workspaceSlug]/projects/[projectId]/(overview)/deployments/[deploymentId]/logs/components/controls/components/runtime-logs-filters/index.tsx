"use client";

import { type FilterItemConfig, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";
import { RuntimeLogsMessageFilter } from "./runtime-logs-message-filter";
import { RuntimeLogsSeverityFilter } from "./runtime-logs-severity-filter";

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "severity",
    label: "Severity",
    shortcut: "S",
    shortcutLabel: "S",
    component: <RuntimeLogsSeverityFilter />,
  },
  {
    id: "message",
    label: "Message",
    shortcut: "M",
    shortcutLabel: "M",
    component: <RuntimeLogsMessageFilter />,
  },
];

export function RuntimeLogsFilters() {
  const { filters } = useRuntimeLogsFilters();

  const filterCount = filters.filter((f) => f.field === "severity" || f.field === "message").length;

  return (
    <FiltersPopover items={FILTER_ITEMS} activeFilters={filters}>
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 rounded-lg",
            filterCount > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
          {filterCount > 0 && (
            <div className="bg-gray-7 rounded h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {filterCount}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
}
