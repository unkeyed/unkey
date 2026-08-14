"use client";

import { createAppEnvironmentSearchItems } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-environment-search-items";
import {
  type AppEnvironmentSelection,
  createAppEnvironmentFilters,
  getAppEnvironmentSelection,
  groupEnvironmentsByApp,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-environment-selection";
import { useAppFilterOptions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  type FilterItemConfig,
  type FilterSearchItem,
  FiltersPopover,
} from "@/components/logs/checkbox/filters-popover";
import type {
  RuntimeLogsFilterField,
  RuntimeLogsFilterValue,
} from "@/lib/schemas/runtime-logs.filter.schema";
import { trpc } from "@/lib/trpc/client";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { RuntimeLogsAppFilter } from "./runtime-logs-app-filter";
import { RuntimeLogsDeploymentFilter } from "./runtime-logs-deployment-filter";
import { RuntimeLogsInstanceFilter } from "./runtime-logs-instance-filter";
import { RuntimeLogsRegionFilter } from "./runtime-logs-region-filter";
import { RuntimeLogsSeverityFilter, severityOptions } from "./runtime-logs-severity-filter";
import { RuntimeLogsTextFilter } from "./runtime-logs-text-filter";

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
    component: <RuntimeLogsTextFilter field="message" label="Message" />,
  },
  {
    id: "attributes",
    label: "Attributes",
    shortcut: "T",
    shortcutLabel: "T",
    component: <RuntimeLogsTextFilter field="attributes" label="Attributes" />,
  },
  {
    id: "appId",
    label: "App",
    shortcut: "A",
    shortcutLabel: "A",
    component: <RuntimeLogsAppFilter />,
  },
  {
    id: "deploymentId",
    label: "Deployment",
    shortcut: "D",
    shortcutLabel: "D",
    component: <RuntimeLogsDeploymentFilter />,
  },
  {
    id: "region",
    label: "Region",
    shortcut: "R",
    shortcutLabel: "R",
    component: <RuntimeLogsRegionFilter />,
  },
  {
    id: "instanceId",
    label: "Instance",
    shortcut: "I",
    shortcutLabel: "I",
    component: <RuntimeLogsInstanceFilter />,
  },
];

export function RuntimeLogsFilters() {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const apps = useAppFilterOptions();
  const { deployments, environments, projectId } = useProjectData();
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  // Region and instance options are only needed once the popover opens; keep
  // the rate-limited queries off the logs page critical path.
  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: isPopoverOpen },
  );
  const { data: instances } = trpc.deploy.runtimeLogs.listInstances.useQuery(
    { projectId },
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
        ...createAppEnvironmentFilters<RuntimeLogsFilterValue>(
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
    (field: RuntimeLogsFilterField, value: string) => {
      const isSelected = filters.some(
        (filter) => filter.field === field && String(filter.value) === value,
      );
      updateFilters(
        isSelected
          ? filters.filter((filter) => filter.field !== field || String(filter.value) !== value)
          : [
              ...filters,
              {
                id: crypto.randomUUID(),
                field,
                operator: "is",
                value,
              },
            ],
      );
    },
    [filters, updateFilters],
  );

  const searchItems = useMemo<FilterSearchItem[]>(() => {
    const isFilterSelected = (field: RuntimeLogsFilterField, value: string) =>
      filters.some((filter) => filter.field === field && String(filter.value) === value);

    const appItems = createAppEnvironmentSearchItems({
      apps,
      environmentsByAppId,
      selection: appEnvironmentSelection,
      onSelectionChange: updateAppEnvironmentSelection,
    });

    const severityItems: FilterSearchItem[] = severityOptions.map((option) => ({
      kind: "option",
      id: `severity:${option.severity}`,
      label: option.label,
      path: ["Severity"],
      checked: isFilterSelected("severity", option.severity),
      onSelect: () => toggleFilter("severity", option.severity),
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

    const regionItems: FilterSearchItem[] = (availableRegions ?? []).map((region) => ({
      kind: "option",
      id: `region:${region.name}`,
      label: region.name,
      path: ["Region"],
      checked: isFilterSelected("region", region.name),
      onSelect: () => toggleFilter("region", region.name),
    }));

    const instanceItems: FilterSearchItem[] = (instances ?? []).map((instance) => ({
      kind: "option",
      id: `instance:${instance.id}`,
      label: instance.id,
      path: ["Instance"],
      description: instance.region.name,
      keywords: [instance.region.name],
      checked: isFilterSelected("instanceId", instance.id),
      onSelect: () => toggleFilter("instanceId", instance.id),
    }));

    return [...appItems, ...severityItems, ...deploymentItems, ...regionItems, ...instanceItems];
  }, [
    appEnvironmentSelection,
    apps,
    availableRegions,
    deployments,
    environmentsByAppId,
    filters,
    instances,
    toggleFilter,
    updateAppEnvironmentSelection,
  ]);

  const filterCount = filters.filter(
    (f) =>
      f.field === "severity" ||
      f.field === "message" ||
      f.field === "attributes" ||
      f.field === "appId" ||
      f.field === "deploymentId" ||
      f.field === "environmentId" ||
      f.field === "region" ||
      f.field === "instanceId",
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
}
