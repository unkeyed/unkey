"use client";

import { Button } from "@unkey/ui";
import { useMemo, useState } from "react";
import { AppEnvironmentList, type AppEnvironmentListProps } from "./app-environment-list";
import {
  type AppEnvironmentSelection,
  createAppEnvironmentUnionFilters,
  getAppEnvironmentSelection,
  groupEnvironmentsByApp,
} from "./app-environment-selection";

type FilterLike = {
  field: string;
  value: string | number;
};

type Props<TFilter extends FilterLike> = Pick<
  AppEnvironmentListProps,
  "apps" | "environments" | "isLoading"
> & {
  filters: TFilter[];
  updateFilters: (filters: TFilter[]) => void;
  createFilter: (field: "appId" | "environmentId", value: string) => TFilter;
  onApply?: () => void;
};

export function AppEnvironmentFilterList<TFilter extends FilterLike>({
  apps,
  environments,
  isLoading,
  filters,
  updateFilters,
  createFilter,
  onApply,
}: Props<TFilter>) {
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

  const applyFilter = () => {
    const appEnvironmentFilters = createAppEnvironmentUnionFilters(
      draft,
      environmentIdsByApp,
      createFilter,
    );
    const otherFilters = filters.filter(
      (filter) => filter.field !== "appId" && filter.field !== "environmentId",
    );

    updateFilters([...otherFilters, ...appEnvironmentFilters]);
    onApply?.();
  };

  return (
    <div className="flex w-80 flex-col overflow-hidden">
      <AppEnvironmentList
        apps={apps}
        environments={environments}
        isLoading={isLoading}
        value={draft}
        onChange={setDraft}
      />

      <div className="border-t p-2">
        <Button variant="primary" className="h-9 w-full rounded-md" onClick={applyFilter}>
          Apply Filter
        </Button>
      </div>
    </div>
  );
}
