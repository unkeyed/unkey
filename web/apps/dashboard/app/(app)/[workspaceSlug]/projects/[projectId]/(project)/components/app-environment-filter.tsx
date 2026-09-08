"use client";

import { useAppFilterOptionsWithLoading } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/components/app-filter-options";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { AppEnvironmentFilterList } from "@/components/deploy/app-environment-filter-list";

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

  return (
    <AppEnvironmentFilterList
      apps={apps}
      environments={environments}
      isLoading={isAppsLoading || isEnvironmentsLoading}
      filters={filters}
      updateFilters={updateFilters}
      createFilter={createFilter}
    />
  );
}
