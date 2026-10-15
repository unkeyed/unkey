import { useFilters } from "@/app/(app)/[workspaceSlug]/logs/hooks/use-filters";
import {
  type FilterItemConfig,
  FiltersPopover,
  createFilterOptionBuilder,
} from "@/components/logs/checkbox/filters-popover";
import type { LogsFilterField, LogsFilterValue } from "@/lib/schemas/logs.filter.schema";
import { IconBarsFilterOutline18 } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo } from "react";
import { MethodsFilter, methodOptions } from "./components/methods-filter";
import { PathsFilter } from "./components/paths-filter";
import { StatusFilter, statusOptions } from "./components/status-filter";

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "E",
    shortcutLabel: "E",
    component: <StatusFilter />,
  },
  {
    id: "methods",
    label: "Method",
    shortcut: "M",
    shortcutLabel: "M",
    component: <MethodsFilter />,
  },
  {
    id: "paths",
    label: "Path",
    shortcut: "P",
    shortcutLabel: "P",
    component: <PathsFilter />,
  },
];

const createFilter = (field: LogsFilterField, value: string): LogsFilterValue => ({
  id: crypto.randomUUID(),
  field,
  operator: "is",
  value: field === "status" ? Number(value) : value,
});

export const LogsFilters = () => {
  const { filters, updateFilters } = useFilters();
  const searchItems = useMemo(() => {
    const option = createFilterOptionBuilder({ filters, updateFilters, createFilter });
    return [
      ...statusOptions.map((status) =>
        option({
          field: "status",
          value: String(status.status),
          label: status.label,
          path: ["Status"],
          description: status.display,
        }),
      ),
      ...methodOptions.map((method) =>
        option({ field: "methods", value: method.method, label: method.method, path: ["Method"] }),
      ),
    ];
  }, [filters, updateFilters]);
  return (
    <FiltersPopover items={FILTER_ITEMS} searchItems={searchItems} activeFilters={filters}>
      <div className="group">
        <Button
          variant="ghost"
          size="md"
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
