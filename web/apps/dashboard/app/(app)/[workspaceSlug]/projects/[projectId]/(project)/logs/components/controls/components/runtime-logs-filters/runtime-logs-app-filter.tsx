"use client";

import { useAppFilterOptions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { CaretRight, Cube, Layers3 } from "@unkey/icons";
import { Button, Checkbox } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useMemo, useState } from "react";
import {
  createAppEnvironmentFilters,
  getAppEnvironmentSelection,
  isEntireAppSelected,
  toggleAppSelection,
  toggleEnvironmentSelection,
} from "./runtime-logs-app-environment-selection";

export const RuntimeLogsAppFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const apps = useAppFilterOptions();
  const { environments } = useProjectData();
  const [expandedAppIds, setExpandedAppIds] = useState<Set<string>>(new Set());
  const [selectedAppIds, setSelectedAppIds] = useState<Set<string>>(new Set());
  const [selectedEnvironmentIds, setSelectedEnvironmentIds] = useState<Set<string>>(new Set());

  const appliedSelection = useMemo(() => getAppEnvironmentSelection(filters), [filters]);

  useEffect(() => {
    setSelectedAppIds(appliedSelection.appIds);
    setSelectedEnvironmentIds(appliedSelection.environmentIds);
  }, [appliedSelection]);

  const environmentsByAppId = useMemo(() => {
    const grouped = new Map<string, typeof environments>();
    for (const environment of environments) {
      const appEnvironments = grouped.get(environment.appId) ?? [];
      appEnvironments.push(environment);
      grouped.set(environment.appId, appEnvironments);
    }
    return grouped;
  }, [environments]);

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

  const sortedApps = useMemo(
    () => apps.toSorted((left, right) => left.name.localeCompare(right.name)),
    [apps],
  );

  const toggleApp = (appId: string, environmentIds: string[]) => {
    const next = toggleAppSelection(
      { appIds: selectedAppIds, environmentIds: selectedEnvironmentIds },
      appId,
      environmentIds,
    );
    setSelectedAppIds(next.appIds);
    setSelectedEnvironmentIds(next.environmentIds);
  };

  const toggleEnvironment = (appId: string, environmentId: string, environmentIds: string[]) => {
    const next = toggleEnvironmentSelection(
      { appIds: selectedAppIds, environmentIds: selectedEnvironmentIds },
      appId,
      environmentId,
      environmentIds,
    );
    setSelectedAppIds(next.appIds);
    setSelectedEnvironmentIds(next.environmentIds);
  };

  const applyFilter = () => {
    const appEnvironmentFilters = createAppEnvironmentFilters(
      { appIds: selectedAppIds, environmentIds: selectedEnvironmentIds },
      environmentIdsByApp,
    );
    const otherFilters = filters.filter(
      (filter) => filter.field !== "appId" && filter.field !== "environmentId",
    );

    updateFilters([...otherFilters, ...appEnvironmentFilters]);
  };

  return (
    <div className="flex w-80 flex-col overflow-hidden">
      <div className="max-h-72 overflow-y-auto p-2 [scrollbar-width:thin]">
        {sortedApps.length === 0 ? (
          <div className="px-2 py-8 text-center text-xs text-gray-9">No apps available</div>
        ) : (
          <div className="flex flex-col gap-1">
            {sortedApps.map((app) => {
              const allAppEnvironments = (environmentsByAppId.get(app.appId) ?? []).toSorted(
                (left, right) => left.slug.localeCompare(right.slug),
              );
              const environmentIds = allAppEnvironments.map((environment) => environment.id);
              const selectedEnvironmentCount = environmentIds.filter((environmentId) =>
                selectedEnvironmentIds.has(environmentId),
              ).length;
              const entireAppSelected = isEntireAppSelected(
                { appIds: selectedAppIds, environmentIds: selectedEnvironmentIds },
                app.appId,
                environmentIds,
              );
              const checkboxState = entireAppSelected
                ? true
                : selectedEnvironmentCount > 0
                  ? "indeterminate"
                  : false;
              const expanded = expandedAppIds.has(app.appId);

              return (
                <div key={app.appId}>
                  <div
                    className={cn(
                      "flex h-9 items-center gap-2 rounded-md px-2",
                      "hover:bg-grayA-3",
                      checkboxState !== false && "bg-grayA-3",
                    )}
                  >
                    <Checkbox
                      checked={checkboxState}
                      onCheckedChange={() => toggleApp(app.appId, environmentIds)}
                      className="size-4 shrink-0 rounded-sm border-gray-5 [&_svg]:size-3"
                      aria-label={`Select all environments in ${app.name}`}
                    />
                    <button
                      type="button"
                      className="flex min-w-0 flex-1 items-center gap-2 text-left outline-hidden"
                      onClick={() =>
                        setExpandedAppIds((current) => {
                          const next = new Set(current);
                          if (next.has(app.appId)) {
                            next.delete(app.appId);
                          } else {
                            next.add(app.appId);
                          }
                          return next;
                        })
                      }
                      aria-expanded={expanded}
                    >
                      <Cube className="size-4 shrink-0 text-accent-9" />
                      <span className="truncate text-xs font-medium text-accent-12">
                        {app.name}
                      </span>
                      <span className="ml-auto shrink-0 text-[11px] text-gray-9">
                        {entireAppSelected
                          ? "All environments"
                          : selectedEnvironmentCount > 0
                            ? `${selectedEnvironmentCount} selected`
                            : `${environmentIds.length} env${environmentIds.length === 1 ? "" : "s"}`}
                      </span>
                      {environmentIds.length > 0 ? (
                        <CaretRight
                          className={cn(
                            "size-3 shrink-0 text-gray-8 transition-transform",
                            expanded && "rotate-90",
                          )}
                        />
                      ) : null}
                    </button>
                  </div>

                  {expanded ? (
                    <div className="ml-[15px] border-l border-grayA-5 py-1 pl-[17px]">
                      {allAppEnvironments.length === 0 ? (
                        <div className="px-2 py-2 text-[11px] text-gray-9">No environments</div>
                      ) : (
                        allAppEnvironments.map((environment) => {
                          const checked =
                            selectedAppIds.has(app.appId) ||
                            selectedEnvironmentIds.has(environment.id);

                          return (
                            <label
                              key={environment.id}
                              htmlFor={`runtime-environment-${environment.id}`}
                              className="flex h-8 cursor-pointer items-center gap-2 rounded-md px-2 hover:bg-grayA-3"
                            >
                              <Checkbox
                                id={`runtime-environment-${environment.id}`}
                                checked={checked}
                                onCheckedChange={() =>
                                  toggleEnvironment(app.appId, environment.id, environmentIds)
                                }
                                className="size-4 shrink-0 rounded-sm border-gray-5 [&_svg]:size-3"
                                aria-label={`Select ${environment.slug} in ${app.name}`}
                              />
                              <Layers3 className="size-3.5 shrink-0 text-accent-9" />
                              <span className="truncate text-xs capitalize text-accent-12">
                                {environment.slug}
                              </span>
                            </label>
                          );
                        })
                      )}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="border-t border-gray-4 p-2">
        <Button variant="primary" className="h-9 w-full rounded-md" onClick={applyFilter}>
          Apply Filter
        </Button>
      </div>
    </div>
  );
};
