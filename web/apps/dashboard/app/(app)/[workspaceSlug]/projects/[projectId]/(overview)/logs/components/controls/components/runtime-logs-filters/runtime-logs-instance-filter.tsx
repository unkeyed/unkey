"use client";

import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { useParams } from "next/navigation";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

type InstanceOption = {
  id: number;
  instanceId: string;
  label: string;
  region: string;
  checked: boolean;
};

export const RuntimeLogsInstanceFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const params = useParams<{ projectId: string }>();
  const { data: instances, isLoading } = trpc.deploy.runtimeLogs.listInstances.useQuery({
    projectId: params.projectId,
  });

  const options: InstanceOption[] = useMemo(
    () =>
      (instances ?? []).map((instance, index) => ({
        id: index,
        instanceId: instance.id,
        label: instance.id,
        region: instance.region.name,
        checked: false,
      })),
    [instances],
  );

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2 p-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="flex items-center gap-4.5 px-2 py-1">
            <div className="size-4 bg-grayA-3 rounded animate-pulse shrink-0" />
            <div className="h-4 w-[48px] bg-grayA-3 rounded animate-pulse" />
            <div className="h-4 w-[120px] bg-grayA-3 rounded animate-pulse" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <FilterCheckbox
      options={options}
      filterField="instanceId"
      checkPath="instanceId"
      selectionMode="multiple"
      renderOptionContent={(option) => (
        <div className="text-accent-12 text-xs flex items-center gap-4.5">
          <span className="text-accent-9">{option.region}</span>
          <span className="font-mono">{option.instanceId}</span>
        </div>
      )}
      createFilterValue={(option) => ({
        value: option.instanceId,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
