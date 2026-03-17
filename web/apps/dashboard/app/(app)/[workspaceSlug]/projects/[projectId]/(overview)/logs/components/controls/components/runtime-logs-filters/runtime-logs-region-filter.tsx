"use client";

import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { RegionFlag } from "../../../../../../components/region-flag";
import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { useMemo } from "react";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

type RegionOption = {
  id: number;
  region: string;
  label: string;
  checked: boolean;
};

export function RuntimeLogsRegionFilter() {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();

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

  return (
    <FilterCheckbox
      options={options}
      filterField="region"
      checkPath="region"
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
