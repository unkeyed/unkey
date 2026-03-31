"use client";

import { collection } from "@/lib/collections";
import type { Environment } from "@/lib/collections/deploy/environments";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import { useCallback, useDeferredValue, useMemo, useState } from "react";
import { EnvVarActionMenu } from "./env-var-action-menu";
import { EnvVarEditRow } from "./env-var-edit-row";
import { EnvVarNameCell } from "./env-var-name-cell";
import { EnvVarValueCell } from "./env-var-value-cell";
import { EnvVarsEmpty } from "./env-vars-empty";
import { EnvVarsSkeleton } from "./env-vars-skeleton";
import type { EnvironmentFilter, SortOption } from "./env-vars-toolbar";

type EnvVarsListProps = {
  projectId: string;
  environments: Environment[];
  searchQuery: string;
  environmentFilter: EnvironmentFilter;
  sortBy: SortOption;
};

export function EnvVarsList({
  projectId,
  environments,
  searchQuery,
  environmentFilter,
  sortBy,
}: EnvVarsListProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const closeEdit = useCallback(() => setEditingId(null), []);
  const deferredQuery = useDeferredValue(searchQuery);

  const { data: envVarData, isLoading } = useLiveQuery(
    (q) => q.from({ v: collection.envVars }).where(({ v }) => eq(v.projectId, projectId)),
    [projectId],
  );

  const envMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const env of environments) {
      map.set(env.id, env.slug);
    }
    return map;
  }, [environments]);

  const indexedVars = useMemo(() => {
    if (!envVarData) {
      return [];
    }
    return envVarData.map((v) => ({
      id: v.id,
      key: v.key,
      keyLower: v.key.toLowerCase(),
      type: v.type,
      environmentId: v.environmentId,
      updatedAt: v.updatedAt,
      note: v.description,
    }));
  }, [envVarData]);

  const filteredData = useMemo(() => {
    const query = deferredQuery.toLowerCase();

    const result = [];
    for (const v of indexedVars) {
      if (query && !v.keyLower.includes(query)) {
        continue;
      }
      if (environmentFilter !== "all" && v.environmentId !== environmentFilter) {
        continue;
      }
      result.push({
        id: v.id,
        key: v.key,
        environmentId: v.environmentId,
        environmentName: envMap.get(v.environmentId) ?? "Unknown",
        type: v.type,
        updatedAt: v.updatedAt,
        note: v.note,
      });
    }

    if (sortBy === "name-asc") {
      result.sort((a, b) => a.key.localeCompare(b.key));
    } else {
      result.sort((a, b) => b.updatedAt - a.updatedAt);
    }

    return result;
  }, [indexedVars, envMap, deferredQuery, environmentFilter, sortBy]);

  if (isLoading) {
    return <EnvVarsSkeleton />;
  }

  if (filteredData.length === 0) {
    return <EnvVarsEmpty searchQuery={searchQuery} />;
  }

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {filteredData.map((item) => (
        <div key={item.id}>
          <div className="group flex items-center hover:bg-grayA-2 transition-colors">
            <div className="flex-3 min-w-0 py-3.5 flex items-center">
              <EnvVarNameCell
                envVarId={item.id}
                variableKey={item.key}
                environmentName={item.environmentName}
                note={item.note}
                searchQuery={deferredQuery}
                type={item.type}
              />
            </div>
            <div className="flex-4 min-w-0 py-3.5 flex items-center pr-3">
              <EnvVarValueCell envVarId={item.id} type={item.type} />
            </div>
            <div className="flex-2 min-w-0 py-3.5 flex items-center pr-3">
              <Badge className="px-1.5 rounded-md flex gap-2 items-center h-5.5 border-none bg-grayA-3 text-grayA-11 truncate">
                <ChartActivity2 iconSize="sm-regular" className="shrink-0" />
                <TimestampInfo displayType="relative" value={item.updatedAt} className="truncate" />
              </Badge>
            </div>
            <div className="w-12 shrink-0 py-3.5 pr-3 flex items-center justify-end">
              <EnvVarActionMenu
                envVarId={item.id}
                variableKey={item.key}
                type={item.type}
                onEdit={() => setEditingId(item.id)}
              />
            </div>
          </div>
          {editingId === item.id && (
            <div className="grid animate-expand-down overflow-hidden">
              <div className="min-h-0">
                <EnvVarEditRow
                  environmentId={item.environmentId}
                  envVarId={item.id}
                  variableKey={item.key}
                  type={item.type}
                  note={item.note ?? null}
                  onClose={closeEdit}
                />
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
