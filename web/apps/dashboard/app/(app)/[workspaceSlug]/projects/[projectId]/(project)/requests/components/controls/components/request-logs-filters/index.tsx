"use client";

import { createAppEnvironmentSearchItems } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-environment-search-items";
import {
  type AppEnvironmentSelection,
  createAppEnvironmentFilters,
  getAppEnvironmentSelection,
  groupEnvironmentsByApp,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-environment-selection";
import { useAppFilterOptions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  type FilterItemConfig,
  type FilterSearchItem,
  FiltersPopover,
} from "@/components/logs/checkbox/filters-popover";
import type {
  RequestLogsFilterField,
  RequestLogsFilterOperator,
  RequestLogsFilterValue,
} from "@/lib/schemas/request-logs.filter.schema";
import { trpc } from "@/lib/trpc/client";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { RequestAppFilter } from "./components/request-logs-app-filter";
import { RequestDeploymentFilter } from "./components/request-logs-deployment-filter";
import {
  RequestMethodsFilter,
  requestMethodOptions,
} from "./components/request-logs-methods-filter";
import { RequestPathsFilter } from "./components/request-logs-paths-filter";
import { RequestStatusFilter, requestStatusOptions } from "./components/request-logs-status-filter";
import { RequestLogsTextFilter } from "./components/request-logs-text-filter";

const EXACT_OPERATOR = ["is"] as const;
const CONTAINS_OPERATOR = ["contains"] as const;

function validateIndexedText(_operator: RequestLogsFilterOperator, value: string): string | null {
  if (value.length < 3) {
    return "Enter at least 3 characters.";
  }
  if (value.length > 512) {
    return "Enter 512 characters or fewer.";
  }
  return null;
}

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
  {
    id: "userAgent",
    label: "User agent",
    shortcut: "U",
    shortcutLabel: "U",
    component: (
      <RequestLogsTextFilter
        field="userAgent"
        label="User agent"
        operators={CONTAINS_OPERATOR}
        validate={validateIndexedText}
      />
    ),
  },
];

export const RequestLogsFilters = () => {
  const { filters, updateFilters } = useRequestLogsFilters();
  const apps = useAppFilterOptions();
  const { customDomains, deployments, domains, environments } = useProjectData();
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: isPopoverOpen },
  );

  const environmentsByAppId = useMemo(() => groupEnvironmentsByApp(environments), [environments]);
  const environmentIdsByApp = useMemo(
    () =>
      new Map(
        apps.map((app) => [
          app.appId,
          (environmentsByAppId.get(app.appId) ?? []).map((environment) => environment.id),
        ]),
      ),
    [apps, environmentsByAppId],
  );
  const appEnvironmentSelection = useMemo(() => getAppEnvironmentSelection(filters), [filters]);

  const updateAppEnvironmentSelection = useCallback(
    (selection: AppEnvironmentSelection) => {
      const otherFilters = filters.filter(
        (filter) => filter.field !== "appId" && filter.field !== "environmentId",
      );
      updateFilters([
        ...otherFilters,
        ...createAppEnvironmentFilters<RequestLogsFilterValue>(
          selection,
          environmentIdsByApp,
          (field, value) => ({
            id: crypto.randomUUID(),
            field,
            operator: "is",
            value,
          }),
        ),
      ]);
    },
    [environmentIdsByApp, filters, updateFilters],
  );

  const toggleFilter = useCallback(
    (
      field: RequestLogsFilterField,
      value: string | number,
      metadata?: RequestLogsFilterValue["metadata"],
    ) => {
      const isSelected = filters.some(
        (filter) => filter.field === field && String(filter.value) === String(value),
      );
      updateFilters(
        isSelected
          ? filters.filter(
              (filter) => filter.field !== field || String(filter.value) !== String(value),
            )
          : [
              ...filters,
              {
                id: crypto.randomUUID(),
                field,
                operator: "is",
                value,
                metadata,
              },
            ],
      );
    },
    [filters, updateFilters],
  );

  const hostnames = useMemo(
    () =>
      [
        ...new Set([
          ...domains.map((domain) => domain.fullyQualifiedDomainName),
          ...customDomains.map((domain) => domain.domain),
        ]),
      ].toSorted(),
    [customDomains, domains],
  );

  const searchItems = useMemo<FilterSearchItem[]>(() => {
    const isFilterSelected = (field: RequestLogsFilterField, value: string | number) =>
      filters.some((filter) => filter.field === field && String(filter.value) === String(value));

    const appItems = createAppEnvironmentSearchItems({
      apps,
      environmentsByAppId,
      selection: appEnvironmentSelection,
      onSelectionChange: updateAppEnvironmentSelection,
    });

    const statusItems: FilterSearchItem[] = requestStatusOptions.map((option) => ({
      kind: "option",
      id: `status:${option.status}`,
      label: `${option.display} ${option.label}`,
      path: ["Status"],
      checked: isFilterSelected("status", option.status),
      onSelect: () => toggleFilter("status", option.status, { colorClass: option.color }),
    }));

    const methodItems: FilterSearchItem[] = requestMethodOptions.map((option) => ({
      kind: "option",
      id: `method:${option.method}`,
      label: option.method,
      path: ["Method"],
      checked: isFilterSelected("methods", option.method),
      onSelect: () => toggleFilter("methods", option.method),
    }));

    const deploymentItems: FilterSearchItem[] = deployments.map((deployment) => ({
      kind: "option",
      id: `deployment:${deployment.id}`,
      label: deployment.gitBranch ?? deployment.id,
      path: ["Deployment"],
      description: deployment.gitBranch ? deployment.id : undefined,
      keywords: [deployment.id, deployment.gitBranch ?? ""],
      checked: isFilterSelected("deploymentId", deployment.id),
      onSelect: () => toggleFilter("deploymentId", deployment.id),
    }));

    const hostnameItems: FilterSearchItem[] = hostnames.map((hostname) => ({
      kind: "option",
      id: `hostname:${hostname}`,
      label: hostname,
      path: ["Hostname"],
      checked: isFilterSelected("host", hostname),
      onSelect: () => toggleFilter("host", hostname),
    }));

    const regionItems: FilterSearchItem[] = (availableRegions ?? []).map((region) => ({
      kind: "option",
      id: `region:${region.name}`,
      label: region.name,
      path: ["Region"],
      checked: isFilterSelected("region", region.name),
      onSelect: () => toggleFilter("region", region.name),
    }));

    return [
      ...appItems,
      ...statusItems,
      ...methodItems,
      ...deploymentItems,
      ...hostnameItems,
      ...regionItems,
    ];
  }, [
    appEnvironmentSelection,
    apps,
    availableRegions,
    deployments,
    environmentsByAppId,
    filters,
    hostnames,
    toggleFilter,
    updateAppEnvironmentSelection,
  ]);

  const filterCount = filters.filter(
    (filter) =>
      filter.field !== "startTime" && filter.field !== "endTime" && filter.field !== "since",
  ).length;

  return (
    <FiltersPopover
      items={FILTER_ITEMS}
      searchItems={searchItems}
      activeFilters={filters}
      open={isPopoverOpen}
      onOpenChange={setIsPopoverOpen}
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
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
          {filterCount > 0 && (
            <div className="bg-gray-7 rounded-sm h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {filterCount}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
