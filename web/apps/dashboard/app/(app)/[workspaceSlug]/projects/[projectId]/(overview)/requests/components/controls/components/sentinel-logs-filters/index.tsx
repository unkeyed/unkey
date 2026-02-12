import { type FilterItemConfig, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useSentinelLogsFilters } from "../../../../hooks/use-sentinel-logs-filters";
import { SentinelDeploymentFilter } from "./components/sentinel-logs-deployment-filter";
import { SentinelEnvironmentFilter } from "./components/sentinel-logs-environment-filter";
import { SentinelMethodsFilter } from "./components/sentinel-logs-methods-filter";
import { SentinelPathsFilter } from "./components/sentinel-logs-paths-filter";
import { SentinelStatusFilter } from "./components/sentinel-logs-status-filter";

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "E",
    shortcutLabel: "E",
    component: <SentinelStatusFilter />,
  },
  {
    id: "methods",
    label: "Method",
    shortcut: "M",
    shortcutLabel: "M",
    component: <SentinelMethodsFilter />,
  },
  {
    id: "paths",
    label: "Path",
    shortcut: "P",
    shortcutLabel: "P",
    component: <SentinelPathsFilter />,
  },
  {
    id: "deploymentId",
    label: "Deployment",
    shortcut: "D",
    shortcutLabel: "D",
    component: <SentinelDeploymentFilter />,
  },
  {
    id: "environmentId",
    label: "Environment",
    shortcut: "N",
    shortcutLabel: "N",
    component: <SentinelEnvironmentFilter />,
  },
];

export const SentinelLogsFilters = () => {
  const { filters } = useSentinelLogsFilters();
  return (
    <FiltersPopover items={FILTER_ITEMS} activeFilters={filters}>
      <div className="group">
        <Button
          variant="ghost"
          size="md"
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
