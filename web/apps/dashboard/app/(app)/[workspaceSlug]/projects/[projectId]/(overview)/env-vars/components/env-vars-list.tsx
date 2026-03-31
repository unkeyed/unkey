"use client";

import { collection } from "@/lib/collections";
import type { Environment } from "@/lib/collections/deploy/environments";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Badge } from "@unkey/ui";
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from "react";
import {
  type DisplayRow,
  type EnvVarItem,
  EnvVarItemRow,
  TimestampBadge,
  groupByKey,
  rowKey,
  rowTime,
} from "./env-var-item-row";
import { EnvVarsEmpty } from "./env-vars-empty";
import { EnvVarsSkeleton } from "./env-vars-skeleton";
import type { EnvironmentFilter, SortOption } from "./env-vars-toolbar";
import { HighlightMatch } from "./highlight-match";

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
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const closeEdit = useCallback(() => setEditingId(null), []);
  const deferredQuery = useDeferredValue(searchQuery);

  const toggleGroup = useCallback((key: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }, []);

  const { data: envVarData, isLoading } = useLiveQuery(
    (q) => q.from({ v: collection.envVars }).where(({ v }) => eq(v.projectId, projectId)),
    [projectId],
  );

  const displayRows = useMemo((): DisplayRow[] => {
    if (!envVarData) {
      return [];
    }

    const query = deferredQuery.toLowerCase();

    const filtered: EnvVarItem[] = [];
    for (const v of envVarData) {
      if (query && !v.key.toLowerCase().includes(query)) {
        continue;
      }
      if (environmentFilter !== "all" && v.environmentId !== environmentFilter) {
        continue;
      }
      filtered.push({
        id: v.id,
        key: v.key,
        environmentId: v.environmentId,
        environmentName: environments.find((e) => e.id === v.environmentId)?.slug ?? "Unknown",
        type: v.type,
        updatedAt: v.updatedAt,
        note: v.description,
      });
    }

    // When filtering by a specific environment, each var is a standalone row.
    // When viewing all environments, group vars that share the same key.
    const rows =
      environmentFilter !== "all"
        ? filtered.map((item): DisplayRow => ({ kind: "single", item }))
        : groupByKey(filtered);

    // Sort rows by name or most recently updated
    if (sortBy === "name-asc") {
      rows.sort((a, b) => rowKey(a).localeCompare(rowKey(b)));
    } else {
      rows.sort((a, b) => rowTime(b) - rowTime(a));
    }

    return rows;
  }, [envVarData, environments, deferredQuery, environmentFilter, sortBy]);

  useCloseEditOnGroupCollapse(displayRows, expandedGroups, editingId, setEditingId);

  if (isLoading) {
    return <EnvVarsSkeleton />;
  }

  if (displayRows.length === 0) {
    return <EnvVarsEmpty searchQuery={searchQuery} />;
  }

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {displayRows.map((row) => {
        if (row.kind === "single") {
          const item = row.item;
          return (
            <EnvVarItemRow
              key={item.id}
              item={item}
              searchQuery={deferredQuery}
              isEditing={editingId === item.id}
              onEdit={() => setEditingId(item.id)}
              onCloseEdit={closeEdit}
            />
          );
        }

        // Group row
        const isExpanded = expandedGroups.has(row.key);
        return (
          <div key={`group-${row.key}`}>
            <div
              //biome-ignore lint/a11y/useSemanticElements: its okay
              role="button"
              tabIndex={0}
              onClick={() => toggleGroup(row.key)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  toggleGroup(row.key);
                }
              }}
              className="group flex items-center hover:bg-grayA-2 transition-colors cursor-pointer"
            >
              <div className="flex-3 min-w-0 py-3.5 flex items-center">
                <div className="flex items-center px-4">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5">
                      <span className="font-mono font-medium text-[13px] text-accent-12 truncate leading-4 max-w-[250px]">
                        <HighlightMatch text={row.key} query={deferredQuery} />
                      </span>
                      {row.hasWriteonly && (
                        <Badge
                          className="px-1.5 py-0 rounded-md h-5 text-[11px] font-medium pointer-events-none"
                          variant="warning"
                        >
                          Sensitive
                        </Badge>
                      )}
                    </div>
                    <div className="text-[13px] mt-1 text-gray-10 capitalize">All Environments</div>
                  </div>
                </div>
              </div>
              <div className="flex-4 min-w-0 py-3.5 flex items-center pr-3">
                <span className="text-[13px] text-gray-9 group-hover:text-gray-11 transition-colors pl-2">
                  {row.items.length} values ›
                </span>
              </div>
              <div className="flex-2 min-w-0 py-3.5 flex items-center pr-3">
                <TimestampBadge value={row.latestUpdatedAt} />
              </div>
              <div className="w-12 shrink-0 py-3.5 pr-3" />
            </div>
            {/* Expanded sub-rows */}
            {isExpanded && (
              <div className="divide-y divide-grayA-3 bg-grayA-2 border-t border-grayA-4">
                {row.items.map((item) => (
                  <EnvVarItemRow
                    key={item.id}
                    item={item}
                    searchQuery={deferredQuery}
                    isEditing={editingId === item.id}
                    onEdit={() => setEditingId(item.id)}
                    onCloseEdit={closeEdit}
                  />
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function useCloseEditOnGroupCollapse(
  displayRows: DisplayRow[],
  expandedGroups: Set<string>,
  editingId: string | null,
  setEditingId: (id: string | null) => void,
) {
  useEffect(() => {
    if (editingId === null) {
      return;
    }
    for (const row of displayRows) {
      if (row.kind === "group" && !expandedGroups.has(row.key)) {
        if (row.items.some((i) => i.id === editingId)) {
          setEditingId(null);
          break;
        }
      }
    }
  }, [expandedGroups, displayRows, editingId, setEditingId]);
}
