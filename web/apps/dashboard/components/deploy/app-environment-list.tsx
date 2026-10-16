"use client";

import type { Environment } from "@/lib/collections/deploy/environments";
import { IconCloudOutline18, IconEyeOutline18 } from "@unkey/icons";
import { Checkbox, QueryView, Skeleton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useId, useMemo } from "react";
import {
  type AppEnvironmentSelection,
  groupEnvironmentsByApp,
  isEntireAppSelected,
  toggleAppSelection,
  toggleEnvironmentSelection,
} from "./app-environment-selection";

export type AppEnvironmentListProps = {
  apps: readonly { appId: string; name: string }[];
  environments: readonly Pick<Environment, "id" | "appId" | "slug" | "kind">[];
  isLoading?: boolean;
  search?: string;
  value: AppEnvironmentSelection;
  onChange: (value: AppEnvironmentSelection) => void;
};

const LOADING_APP_IDS = ["first", "second"];
const ENVIRONMENT_ICONS = {
  production: IconCloudOutline18,
  preview: IconEyeOutline18,
} satisfies Record<Environment["kind"], typeof IconCloudOutline18>;

export function AppEnvironmentList({
  apps,
  environments,
  isLoading,
  search = "",
  value,
  onChange,
}: AppEnvironmentListProps) {
  const id = useId();
  const environmentsByAppId = useMemo(() => groupEnvironmentsByApp(environments), [environments]);
  const sortedApps = useMemo(
    () => apps.toSorted((left, right) => left.name.localeCompare(right.name)),
    [apps],
  );
  const term = search.trim().toLowerCase();
  const visibleApps = sortedApps.filter(
    (app) =>
      app.name.toLowerCase().includes(term) ||
      (environmentsByAppId.get(app.appId) ?? []).some((environment) =>
        `${app.name} ${environment.slug}`.toLowerCase().includes(term),
      ),
  );

  return (
    <div className="max-h-72 min-h-40 overflow-y-auto p-2 [scrollbar-width:thin]">
      <QueryView
        isLoading={isLoading ?? false}
        data={visibleApps}
        loading={
          <div className="flex flex-col gap-1">
            {LOADING_APP_IDS.map((id) => (
              <div key={id}>
                <div className="flex h-9 items-center gap-2 px-2">
                  <Skeleton className="size-4 shrink-0 rounded" />
                  <Skeleton className="h-4 w-24 rounded" />
                </div>
                <div className="ml-[15.5px] border-l py-1 pl-3">
                  <div className="flex h-8 items-center gap-2 px-2">
                    <Skeleton className="size-4 shrink-0 rounded" />
                    <Skeleton className="h-4 w-16 rounded" />
                  </div>
                  <div className="flex h-8 items-center gap-2 px-2">
                    <Skeleton className="size-4 shrink-0 rounded" />
                    <Skeleton className="h-4 w-16 rounded" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        }
        empty={
          <div className="px-2 py-8 text-center text-xs text-gray-9">
            {term ? "No environments match this filter." : "No apps available"}
          </div>
        }
      >
        {(visibleApps) => (
          <div className="flex flex-col gap-1">
            {visibleApps.map((app) => {
              const allAppEnvironments = environmentsByAppId.get(app.appId) ?? [];
              const environmentIds = allAppEnvironments.map((environment) => environment.id);
              const selectedEnvironmentCount = environmentIds.filter((environmentId) =>
                value.environmentIds.has(environmentId),
              ).length;
              const entireAppSelected = isEntireAppSelected(value, app.appId, environmentIds);
              const checkboxState = entireAppSelected
                ? true
                : selectedEnvironmentCount > 0
                  ? "indeterminate"
                  : false;

              return (
                <div key={app.appId}>
                  <label
                    htmlFor={`${id}-app-${app.appId}`}
                    className={cn(
                      "flex h-9 cursor-pointer items-center gap-2 rounded-md px-2 hover:bg-grayA-3",
                      checkboxState !== false && "bg-grayA-3",
                    )}
                  >
                    <Checkbox
                      id={`${id}-app-${app.appId}`}
                      checked={checkboxState}
                      onCheckedChange={() =>
                        onChange(toggleAppSelection(value, app.appId, environmentIds))
                      }
                      className="size-4 shrink-0 rounded-sm [&_svg]:size-3"
                      aria-label={`Select all environments in ${app.name}`}
                    />
                    <span className="truncate text-xs font-medium text-gray-12">{app.name}</span>
                  </label>

                  {allAppEnvironments.length > 0 ? (
                    <div className="ml-[15.5px] border-l py-1 pl-3">
                      {allAppEnvironments.map((environment) => {
                        const checked =
                          value.appIds.has(app.appId) || value.environmentIds.has(environment.id);
                        const Icon = ENVIRONMENT_ICONS[environment.kind];

                        return (
                          <label
                            key={environment.id}
                            htmlFor={`${id}-environment-${environment.id}`}
                            className="flex h-8 cursor-pointer items-center gap-2 rounded-md px-2 hover:bg-grayA-3"
                          >
                            <Checkbox
                              id={`${id}-environment-${environment.id}`}
                              checked={checked}
                              onCheckedChange={() =>
                                onChange(
                                  toggleEnvironmentSelection(
                                    value,
                                    app.appId,
                                    environment.id,
                                    environmentIds,
                                  ),
                                )
                              }
                              className="size-4 shrink-0 rounded-sm [&_svg]:size-3"
                              aria-label={`Select ${environment.slug} in ${app.name}`}
                            />
                            <Icon className="size-3.5 shrink-0 text-gray-11" aria-hidden="true" />
                            <span className="truncate text-xs capitalize text-gray-12">
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
      </QueryView>
    </div>
  );
}
