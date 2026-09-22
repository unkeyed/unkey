"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { trpc } from "@/lib/trpc/client";
import { Skeleton } from "@unkey/ui";
import { useParams } from "next/navigation";
import { useMemo } from "react";

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
          // biome-ignore lint/suspicious/noArrayIndexKey: safe to leave
          <div key={i} className="flex items-center gap-4.5 px-2 py-1">
            <Skeleton className="size-4 rounded shrink-0" />
            <Skeleton className="h-4 w-[48px] rounded" />
            <Skeleton className="h-4 w-[120px] rounded" />
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
        <div className="text-gray-12 text-xs flex items-center gap-4.5">
          <span className="text-gray-9">{option.region}</span>
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
