import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { type FilterItemConfig, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { RequestAppFilter } from "./components/request-logs-app-filter";
import { RequestDeploymentFilter } from "./components/request-logs-deployment-filter";
import { RequestEnvironmentFilter } from "./components/request-logs-environment-filter";
import { RequestMethodsFilter } from "./components/request-logs-methods-filter";
import { RequestPathsFilter } from "./components/request-logs-paths-filter";
import { RequestStatusFilter } from "./components/request-logs-status-filter";

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "E",
    shortcutLabel: "E",
    component: <RequestStatusFilter />,
  },
  {
    id: "methods",
    label: "Method",
    shortcut: "M",
    shortcutLabel: "M",
    component: <RequestMethodsFilter />,
  },
  {
    id: "paths",
    label: "Path",
    shortcut: "P",
    shortcutLabel: "P",
    component: <RequestPathsFilter />,
  },
  {
    id: "appId",
    label: "App",
    shortcut: "A",
    shortcutLabel: "A",
    component: <RequestAppFilter />,
  },
  {
    id: "deploymentId",
    label: "Deployment",
    shortcut: "D",
    shortcutLabel: "D",
    component: <RequestDeploymentFilter />,
  },
  {
    id: "environmentId",
    label: "Environment",
    shortcut: "N",
    shortcutLabel: "N",
    component: <RequestEnvironmentFilter />,
  },
];

export const RequestLogsFilters = () => {
  const { filters } = useRequestLogsFilters();
  return (
    <FiltersPopover items={FILTER_ITEMS} activeFilters={filters}>
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
