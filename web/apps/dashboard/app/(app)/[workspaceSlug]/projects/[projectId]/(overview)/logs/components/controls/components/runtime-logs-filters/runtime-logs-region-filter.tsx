"use client";

import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { useMemo } from "react";
import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/region-flag";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

type RegionOption = {
  id: number;
  region: string;
  label: string;
  checked: boolean;
};

export function RuntimeLogsRegionFilter() {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const { data: availableRegions, isLoading } =
    trpc.deploy.environmentSettings.getAvailableRegions.useQuery();

  const options: RegionOption[] = useMemo(
    () =>
      (availableRegions ?? []).map((r, index) => ({
        id: index,
        region: r.name,
        label: r.name,
        checked: false,
      })),
    [availableRegions],
  );

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2 p-2">
        {Array.from({ length: 3 }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: safe to leave
          <div key={i} className="flex items-center gap-4.5 px-2 py-1">
            <div className="size-4 bg-grayA-3 rounded animate-pulse shrink-0" />
            <div className="size-4 bg-grayA-3 rounded-full animate-pulse shrink-0" />
            <div className="h-4 w-[80px] bg-grayA-3 rounded animate-pulse" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <FilterCheckbox
      options={options}
      filterField="region"
      checkPath="region"
      selectionMode="multiple"
      renderOptionContent={(option) => (
        <>
          <RegionFlag
            flagCode={mapRegionToFlag(option.region)}
            size="xs"
            shape="circle"
            className="[&_img]:size-3"
          />
          <span className="text-accent-12 text-xs font-mono">{option.label}</span>
        </>
      )}
      createFilterValue={(option) => ({
        value: option.region,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
}
