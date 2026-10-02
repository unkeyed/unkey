"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  type AppEnvironmentSelection,
  createAppEnvironmentUnionFilters,
  getAppEnvironmentSelection,
  groupEnvironmentsByApp,
  isEntireAppSelected,
  toggleAppSelection,
  toggleEnvironmentSelection,
} from "@/components/deploy/app-environment-selection";
import type { FilterSearchItem } from "@/components/logs/checkbox/filters-popover";
import { useCallback, useMemo } from "react";
import { useAppFilterOptions } from "./app-filter-options";

type FilterLike = {
  field: string;
  value: string | number;
};

type UseAppEnvironmentSearchItemsParams<TFilter extends FilterLike> = {
  filters: TFilter[];
  updateFilters: (filters: TFilter[]) => void;
  createFilter: (field: "appId" | "environmentId", value: string) => TFilter;
};

export function useAppEnvironmentSearchItems<TFilter extends FilterLike>({
  filters,
  updateFilters,
  createFilter,
}: UseAppEnvironmentSearchItemsParams<TFilter>): FilterSearchItem[] {
  const apps = useAppFilterOptions();
  const { environments } = useProjectData();
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
  const selection = useMemo(() => getAppEnvironmentSelection(filters), [filters]);

  const updateSelection = useCallback(
    (nextSelection: AppEnvironmentSelection) => {
      const otherFilters = filters.filter(
        (filter) => filter.field !== "appId" && filter.field !== "environmentId",
      );
      updateFilters([
        ...otherFilters,
        ...createAppEnvironmentUnionFilters(nextSelection, environmentIdsByApp, createFilter),
      ]);
    },
    [createFilter, environmentIdsByApp, filters, updateFilters],
  );

  return useMemo(
    () =>
      apps.flatMap((app) => {
        const appEnvironments = environmentsByAppId.get(app.appId) ?? [];
        const environmentIds = appEnvironments.map((environment) => environment.id);

        return [
          {
            kind: "option" as const,
            id: `app:${app.appId}`,
            label: app.name,
            path: ["App"],
            keywords: [app.appId, "all environments"],
            checked: isEntireAppSelected(selection, app.appId, environmentIds),
            onSelect: () =>
              updateSelection(toggleAppSelection(selection, app.appId, environmentIds)),
          },
          ...appEnvironments.map(
            (environment): FilterSearchItem => ({
              kind: "option",
              id: `environment:${environment.id}`,
              label: environment.slug,
              path: ["App", app.name],
              keywords: [environment.id, app.appId, app.name],
              checked:
                selection.appIds.has(app.appId) || selection.environmentIds.has(environment.id),
              onSelect: () =>
                updateSelection(
                  toggleEnvironmentSelection(selection, app.appId, environment.id, environmentIds),
                ),
            }),
          ),
        ];
      }),
    [apps, environmentsByAppId, selection, updateSelection],
  );
}
