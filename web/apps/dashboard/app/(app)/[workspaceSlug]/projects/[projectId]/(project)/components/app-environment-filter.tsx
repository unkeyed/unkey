"use client";

import { Button, Checkbox } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import { useAppFilterOptionsWithLoading } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  type AppEnvironmentSelection,
  createAppEnvironmentFilters,
  getAppEnvironmentSelection,
  groupEnvironmentsByApp,
  isEntireAppSelected,
  toggleAppSelection,
  toggleEnvironmentSelection,
} from "./app-environment-selection";

type FilterLike = {
  field: string;
  value: string | number;
};

type AppEnvironmentFilterProps<TFilter extends FilterLike> = {
  filters: TFilter[];
  updateFilters: (filters: TFilter[]) => void;
  createFilter: (field: "appId" | "environmentId", value: string) => TFilter;
};

export function AppEnvironmentFilter<TFilter extends FilterLike>({
  filters,
  updateFilters,
  createFilter,
}: AppEnvironmentFilterProps<TFilter>) {
  const { options: apps, isLoading: isAppsLoading } = useAppFilterOptionsWithLoading();
  const { environments, isEnvironmentsLoading } = useProjectData();
  const isLoading = isAppsLoading || isEnvironmentsLoading;
  // The panel unmounts when its drover closes, so lazy init re-syncs the
  // draft from the applied filters on every open.
  const [draft, setDraft] = useState<AppEnvironmentSelection>(() =>
    getAppEnvironmentSelection(filters),
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

  const sortedApps = useMemo(
    () => apps.toSorted((left, right) => left.name.localeCompare(right.name)),
    [apps],
  );

  const applyFilter = () => {
    const appEnvironmentFilters = createAppEnvironmentFilters(
      draft,
      environmentIdsByApp,
      createFilter,
    );
    const otherFilters = filters.filter(
      (filter) => filter.field !== "appId" && filter.field !== "environmentId",
    );

    updateFilters([...otherFilters, ...appEnvironmentFilters]);
  };

  return (
    <div className="flex w-80 flex-col overflow-hidden">
      <div className="max-h-72 min-h-40 overflow-y-auto p-2 [scrollbar-width:thin]">
        {isLoading ? (
          <div className="flex flex-col gap-1">
            {Array.from({ length: 2 }).map((_, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: safe to leave
              <div key={i}>
                <div className="flex h-9 items-center gap-2 px-2">
                  <div className="size-4 shrink-0 animate-pulse rounded bg-grayA-3" />
                  <div className="h-4 w-24 animate-pulse rounded bg-grayA-3" />
                </div>
                <div className="ml-[15.5px] border-l border-grayA-5 py-1 pl-3">
                  <div className="flex h-8 items-center gap-2 px-2">
                    <div className="size-4 shrink-0 animate-pulse rounded bg-grayA-3" />
                    <div className="h-4 w-16 animate-pulse rounded bg-grayA-3" />
                  </div>
                  <div className="flex h-8 items-center gap-2 px-2">
                    <div className="size-4 shrink-0 animate-pulse rounded bg-grayA-3" />
                    <div className="h-4 w-16 animate-pulse rounded bg-grayA-3" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : sortedApps.length === 0 ? (
          <div className="px-2 py-8 text-center text-xs text-gray-9">No apps available</div>
        ) : (
          <div className="flex flex-col gap-1">
            {sortedApps.map((app) => {
              const allAppEnvironments = environmentsByAppId.get(app.appId) ?? [];
              const environmentIds = allAppEnvironments.map((environment) => environment.id);
              const selectedEnvironmentCount = environmentIds.filter((environmentId) =>
                draft.environmentIds.has(environmentId),
              ).length;
              const entireAppSelected = isEntireAppSelected(draft, app.appId, environmentIds);
              const checkboxState = entireAppSelected
                ? true
                : selectedEnvironmentCount > 0
                  ? "indeterminate"
                  : false;

              return (
                <div key={app.appId}>
                  <label
                    htmlFor={`app-filter-${app.appId}`}
                    className={cn(
                      "flex h-9 cursor-pointer items-center gap-2 rounded-md px-2 hover:bg-grayA-3",
                      checkboxState !== false && "bg-grayA-3",
                    )}
                  >
                    <Checkbox
                      id={`app-filter-${app.appId}`}
                      checked={checkboxState}
                      onCheckedChange={() =>
                        setDraft((current) =>
                          toggleAppSelection(current, app.appId, environmentIds),
                        )
                      }
                      className="size-4 shrink-0 rounded-sm border-gray-5 [&_svg]:size-3"
                      aria-label={`Select all environments in ${app.name}`}
                    />
                    <span className="truncate text-xs font-medium text-accent-12">{app.name}</span>
                  </label>

                  {allAppEnvironments.length > 0 ? (
                    <div className="ml-[15.5px] border-l border-grayA-5 py-1 pl-3">
                      {allAppEnvironments.map((environment) => {
                        const checked =
                          draft.appIds.has(app.appId) || draft.environmentIds.has(environment.id);

                        return (
                          <label
                            key={environment.id}
                            htmlFor={`environment-filter-${environment.id}`}
                            className="flex h-8 cursor-pointer items-center gap-2 rounded-md px-2 hover:bg-grayA-3"
                          >
                            <Checkbox
                              id={`environment-filter-${environment.id}`}
                              checked={checked}
                              onCheckedChange={() =>
                                setDraft((current) =>
                                  toggleEnvironmentSelection(
                                    current,
                                    app.appId,
                                    environment.id,
                                    environmentIds,
                                  ),
                                )
                              }
                              className="size-4 shrink-0 rounded-sm border-gray-5 [&_svg]:size-3"
                              aria-label={`Select ${environment.slug} in ${app.name}`}
                            />
                            <span className="truncate text-xs capitalize text-accent-12">
                              {environment.slug}
                            </span>
                          </label>
                        );
                      })}
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
}
