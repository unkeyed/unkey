"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/(deploy-data)/logs/hooks/use-runtime-logs-filters";
import { AppEnvironmentFilter } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-environment-filter";
import { useAppEnvironmentSearchItems } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/use-app-environment-search-items";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useRecentDeployments } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/hooks/use-recent-deployments";
import {
  type FilterItemConfig,
  type FilterItemOption,
  FiltersPopover,
} from "@/components/logs/checkbox/filters-popover";
import type {
  RuntimeLogsFilterField,
  RuntimeLogsFilterValue,
} from "@/lib/schemas/runtime-logs.filter.schema";
import { trpc } from "@/lib/trpc/client";
import { IconBarsFilterOutline18 } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import { RuntimeLogsDeploymentFilter } from "./runtime-logs-deployment-filter";
import { RuntimeLogsInstanceFilter } from "./runtime-logs-instance-filter";
import { RuntimeLogsRegionFilter } from "./runtime-logs-region-filter";
import { RuntimeLogsSeverityFilter, severityOptions } from "./runtime-logs-severity-filter";
import { RuntimeLogsTextFilter } from "./runtime-logs-text-filter";

const createRuntimeLogsFilter = (
  field: RuntimeLogsFilterField,
  value: string,
): RuntimeLogsFilterValue => ({
  id: crypto.randomUUID(),
  field,
  operator: "is",
  value,
});

function RuntimeLogsAppFilter() {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  return (
    <AppEnvironmentFilter
      filters={filters}
      updateFilters={updateFilters}
      createFilter={createRuntimeLogsFilter}
    />
  );
}

const FILTER_ITEMS: FilterItemConfig<RuntimeLogsFilterField>[] = [
  {
    id: "severity",
    label: "Severity",
    shortcut: "S",
    shortcutLabel: "S",
    component: <RuntimeLogsSeverityFilter />,
    options: severityOptions.map((severity) => ({
      value: severity.severity,
      label: severity.label,
    })),
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
  const { projectId } = useProjectData();
  const { deployments } = useRecentDeployments();
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
  const appItems = useAppEnvironmentSearchItems({
    filters,
    updateFilters,
    createFilter: createRuntimeLogsFilter,
  });

  const items = useMemo(() => {
    const queriedOptions: Partial<Record<RuntimeLogsFilterField, FilterItemOption<string>[]>> = {
      deploymentId: deployments.map((deployment) => ({
        value: deployment.id,
        label: deployment.gitBranch ?? deployment.id,
        description: deployment.gitBranch ? deployment.id : undefined,
        keywords: [deployment.id, deployment.gitBranch ?? ""],
      })),
      region: (availableRegions ?? []).map((region) => ({
        value: region.name,
        label: region.name,
      })),
      instanceId: (instances ?? []).map((instance) => ({
        value: instance.id,
        label: instance.id,
        description: instance.region.name,
        keywords: [instance.region.name],
      })),
    };
    return FILTER_ITEMS.map((item) => ({
      ...item,
      options: item.options ?? queriedOptions[item.id],
    }));
  }, [availableRegions, deployments, instances]);

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
      items={items}
      searchItems={appItems}
      activeFilters={filters}
      updateFilters={updateFilters}
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
          <IconBarsFilterOutline18 className="text-gray-9 size-4" />
          <span className="text-gray-12 font-medium text-sm">Filter</span>
          {filterCount > 0 && <Badge variant="count">{filterCount}</Badge>}
        </Button>
      </div>
    </FiltersPopover>
  );
}
