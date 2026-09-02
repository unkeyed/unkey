"use client";

import { useAppEnvironmentSearchItems } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/use-app-environment-search-items";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { type FilterItemConfig, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import type { RequestLogsFilterValue } from "@/lib/schemas/request-logs.filter.schema";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { IconBarsFilterOutline18 } from "nucleo-ui-outline-18";
import { RequestAppFilter } from "./components/request-logs-app-filter";
import { RequestDeploymentFilter } from "./components/request-logs-deployment-filter";
import { RequestMethodsFilter } from "./components/request-logs-methods-filter";
import { RequestPathsFilter } from "./components/request-logs-paths-filter";
import { RequestStatusFilter } from "./components/request-logs-status-filter";
import { RequestLogsTextFilter } from "./components/request-logs-text-filter";

const EXACT_OPERATOR = ["is"] as const;

const createAppFilter = (
  field: "appId" | "environmentId",
  value: string,
): RequestLogsFilterValue => ({
  id: crypto.randomUUID(),
  field,
  operator: "is",
  value,
});

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "S",
    shortcutLabel: "S",
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
    id: "host",
    label: "Hostname",
    shortcut: "H",
    shortcutLabel: "H",
    component: <RequestLogsTextFilter field="host" label="Hostname" operators={EXACT_OPERATOR} />,
  },
  {
    id: "region",
    label: "Region",
    shortcut: "R",
    shortcutLabel: "R",
    component: <RequestLogsTextFilter field="region" label="Region" operators={EXACT_OPERATOR} />,
  },
  {
    id: "requestId",
    label: "Request ID",
    shortcut: "Q",
    shortcutLabel: "Q",
    component: (
      <RequestLogsTextFilter field="requestId" label="Request ID" operators={EXACT_OPERATOR} />
    ),
  },
];

export const RequestLogsFilters = () => {
  const { filters, updateFilters } = useRequestLogsFilters();
  const appItems = useAppEnvironmentSearchItems({
    filters,
    updateFilters,
    createFilter: createAppFilter,
  });

  const filterCount = filters.filter(
    (filter) =>
      filter.field !== "startTime" && filter.field !== "endTime" && filter.field !== "since",
  ).length;

  return (
    <FiltersPopover
      items={FILTER_ITEMS}
      searchItems={appItems}
      activeFilters={filters}
      getFilterCount={(field) =>
        filters.filter(
          (filter) =>
            filter.field === field || (field === "appId" && filter.field === "environmentId"),
        ).length
      }
    >
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn(
            "group-data-popup-open:bg-gray-4 px-2 rounded-lg",
            filterCount > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <IconBarsFilterOutline18 className="text-accent-9 size-4" />
          <span className="text-accent-12 font-normal text-[13px]">Filter</span>
          {filterCount > 0 && (
            <div className="bg-gray-7 rounded-sm h-4 px-1 text-[11px] font-normal text-accent-12 text-center flex items-center justify-center">
              {filterCount}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
