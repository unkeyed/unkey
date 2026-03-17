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
  const { data: instances } = trpc.deploy.runtimeLogs.listInstances.useQuery({
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

  return (
    <FilterCheckbox
      options={options}
      filterField="instanceId"
      checkPath="instanceId"
      selectionMode="multiple"
      renderOptionContent={(option) => (
        <div className="text-accent-12 text-xs flex items-center gap-4.5">
          <span className="font-mono">{option.instanceId.slice(0, 12)}</span>
          <span className="text-accent-9 ml-1">{option.region}</span>
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
