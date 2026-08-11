"use client";

import { useAppFilterOptions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  type FilterItemConfig,
  type FilterSearchItem,
  FiltersPopover,
} from "@/components/logs/checkbox/filters-popover";
import type { RuntimeLogsFilterField } from "@/lib/schemas/runtime-logs.filter.schema";
import { trpc } from "@/lib/trpc/client";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useMemo } from "react";
import {
  type AppEnvironmentSelection,
  createAppEnvironmentFilters,
  getAppEnvironmentSelection,
  isEntireAppSelected,
  toggleAppSelection,
  toggleEnvironmentSelection,
} from "./runtime-logs-app-environment-selection";
import { RuntimeLogsAppFilter } from "./runtime-logs-app-filter";
import { RuntimeLogsDeploymentFilter } from "./runtime-logs-deployment-filter";
import { RuntimeLogsInstanceFilter } from "./runtime-logs-instance-filter";
import { RuntimeLogsMessageFilter } from "./runtime-logs-message-filter";
import { RuntimeLogsRegionFilter } from "./runtime-logs-region-filter";
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
  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const { data: instances } = trpc.deploy.runtimeLogs.listInstances.useQuery({ projectId });

  const environmentIdsByApp = useMemo(() => {
    const grouped = new Map<string, string[]>();
    for (const app of apps) {
      grouped.set(
        app.appId,
        environments
          .filter((environment) => environment.appId === app.appId)
          .map((environment) => environment.id),
      );
    }
    return grouped;
  }, [apps, environments]);

  const appEnvironmentSelection = useMemo(() => getAppEnvironmentSelection(filters), [filters]);

  const updateAppEnvironmentSelection = useCallback(
    (selection: AppEnvironmentSelection) => {
      const otherFilters = filters.filter(
        (filter) => filter.field !== "appId" && filter.field !== "environmentId",
      );
      updateFilters([
        ...otherFilters,
        ...createAppEnvironmentFilters(selection, environmentIdsByApp),
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

    const appItems: FilterSearchItem[] = apps.flatMap((app) => {
      const appEnvironments = environments
        .filter((environment) => environment.appId === app.appId)
        .toSorted((left, right) => left.slug.localeCompare(right.slug));
      const environmentIds = appEnvironments.map((environment) => environment.id);

      return [
        {
          kind: "option",
          id: `app:${app.appId}`,
          label: app.name,
          path: ["App"],
          keywords: [app.appId, "all environments"],
          checked: isEntireAppSelected(appEnvironmentSelection, app.appId, environmentIds),
          onSelect: () =>
            updateAppEnvironmentSelection(
              toggleAppSelection(appEnvironmentSelection, app.appId, environmentIds),
            ),
        },
        ...appEnvironments.map(
          (environment): FilterSearchItem => ({
            kind: "option",
            id: `environment:${environment.id}`,
            label: environment.slug,
            path: ["App", app.name],
            keywords: [environment.id, app.appId, app.name],
            checked:
              appEnvironmentSelection.appIds.has(app.appId) ||
              appEnvironmentSelection.environmentIds.has(environment.id),
            onSelect: () =>
              updateAppEnvironmentSelection(
                toggleEnvironmentSelection(
                  appEnvironmentSelection,
                  app.appId,
                  environment.id,
                  environmentIds,
                ),
              ),
          }),
        ),
      ];
    });

    const severityItems: FilterSearchItem[] = ["ERROR", "WARN", "INFO", "DEBUG"].map(
      (severity) => ({
        kind: "option",
        id: `severity:${severity}`,
        label: severity.charAt(0) + severity.slice(1).toLowerCase(),
        path: ["Severity"],
        keywords: severity === "WARN" ? ["warning"] : undefined,
        checked: isFilterSelected("severity", severity),
        onSelect: () => toggleFilter("severity", severity),
      }),
    );

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
    environments,
    filters,
    instances,
    toggleFilter,
    updateAppEnvironmentSelection,
  ]);

  const filterCount = filters.filter(
    (f) =>
      f.field === "severity" ||
      f.field === "message" ||
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
