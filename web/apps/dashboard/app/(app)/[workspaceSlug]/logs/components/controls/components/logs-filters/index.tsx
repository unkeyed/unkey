import { useFilters } from "@/app/(app)/[workspaceSlug]/logs/hooks/use-filters";
import { type FilterItemConfig, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import type { LogsFilterField } from "@/lib/schemas/logs.filter.schema";
import { IconBarsFilterOutline18 } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { MethodsFilter, methodOptions } from "./components/methods-filter";
import { PathsFilter } from "./components/paths-filter";
import { StatusFilter, statusOptions } from "./components/status-filter";

const FILTER_ITEMS: FilterItemConfig<LogsFilterField>[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "E",
    shortcutLabel: "E",
    component: <StatusFilter />,
    options: statusOptions.map((status) => ({
      value: status.status,
      label: status.label,
      description: status.display,
    })),
  },
  {
    id: "methods",
    label: "Method",
    shortcut: "M",
    shortcutLabel: "M",
    component: <MethodsFilter />,
    options: methodOptions.map((method) => ({ value: method.method, label: method.method })),
  },
  {
    id: "paths",
    label: "Path",
    shortcut: "P",
    shortcutLabel: "P",
    component: <PathsFilter />,
  },
];

export const LogsFilters = () => {
  const { filters, updateFilters } = useFilters();
  return (
    <FiltersPopover items={FILTER_ITEMS} activeFilters={filters} updateFilters={updateFilters}>
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
