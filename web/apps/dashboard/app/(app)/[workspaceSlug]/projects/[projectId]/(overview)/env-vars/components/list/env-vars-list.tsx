"use client";

import { collection } from "@/lib/collections";
import type { Environment } from "@/lib/collections/deploy/environments";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from "react";
import { useRowSelection } from "../../hooks/use-row-selection";
import { useVirtualList } from "../../hooks/use-virtual-list";
import { EnvVarsEmpty } from "../shared/env-vars-empty";
import { EnvVarsSkeleton } from "../shared/env-vars-skeleton";
import type { EnvironmentFilter, SortOption } from "../toolbar/env-vars-toolbar";
import { GroupRow } from "./env-var-group-row";
import {
  type DisplayRow,
  type EnvVarItem,
  EnvVarItemRow,
  groupByKey,
  rowKey,
  rowTime,
} from "./env-var-item-row";
import { EnvVarSelectionBar } from "./env-var-selection-bar";

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
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const closeEdit = useCallback(() => setEditingId(null), []);
  const deferredQuery = useDeferredValue(searchQuery);

  const toggleGroup = useCallback((key: string) => {
    setExpandedRow((prev) => (prev === key ? "" : key));
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

  const {
    selectedIds,
    toggleRowSelection,
    toggleItemSelection,
    isRowSelected,
    handleBulkDelete,
    clearSelection,
  } = useRowSelection(displayRows);

  useCloseEditOnGroupCollapse(displayRows, expandedRow, editingId, setEditingId);

  const { virtualizer, listRefCallback, scrollMargin } = useVirtualList(
    displayRows,
    editingId,
    expandedRow,
  );

  if (isLoading) {
    return <EnvVarsSkeleton />;
  }

  if (displayRows.length === 0) {
    return <EnvVarsEmpty searchQuery={searchQuery} />;
  }

  const virtualItems = virtualizer.getVirtualItems();

  return (
    <>
      <div ref={listRefCallback} className="border border-grayA-4 rounded-[14px] overflow-hidden">
        <div
          style={{
            height: virtualizer.getTotalSize(),
            position: "relative",
            width: "100%",
          }}
        >
          {virtualItems.map((virtualRow) => {
            const row = displayRows[virtualRow.index];
            const isLast = virtualRow.index === displayRows.length - 1;

            return (
              <div
                key={virtualRow.key}
                ref={virtualizer.measureElement}
                data-index={virtualRow.index}
                className={isLast ? undefined : "border-b border-grayA-4"}
                style={{
                  position: "absolute",
                  top: 0,
                  left: 0,
                  width: "100%",
                  transform: `translateY(${virtualRow.start - scrollMargin}px)`,
                }}
              >
                {row.kind === "single" ? (
                  <EnvVarItemRow
                    item={row.item}
                    searchQuery={deferredQuery}
                    isEditing={editingId === row.item.id}
                    onEdit={() => setEditingId(row.item.id)}
                    onCloseEdit={closeEdit}
                    isSelected={selectedIds.has(row.item.id)}
                    onToggleSelection={(shiftKey) => toggleRowSelection(virtualRow.index, shiftKey)}
                    hasSelection={selectedIds.size > 0}
                  />
                ) : (
                  <GroupRow
                    row={row}
                    isExpanded={expandedRow === row.key}
                    selected={isRowSelected(row)}
                    selectedIds={selectedIds}
                    deferredQuery={deferredQuery}
                    editingId={editingId}
                    onToggleGroup={() => toggleGroup(row.key)}
                    onToggleSelection={(shiftKey) => toggleRowSelection(virtualRow.index, shiftKey)}
                    onToggleItemSelection={toggleItemSelection}
                    onEdit={setEditingId}
                    onCloseEdit={closeEdit}
                    hasSelection={selectedIds.size > 0}
                  />
                )}
              </div>
            );
          })}
        </div>
      </div>
      <EnvVarSelectionBar
        selectedCount={selectedIds.size}
        onDelete={handleBulkDelete}
        onClearSelection={clearSelection}
      />
    </>
  );
}

function useCloseEditOnGroupCollapse(
  displayRows: DisplayRow[],
  expandedRow: string | null,
  editingId: string | null,
  setEditingId: (id: string | null) => void,
) {
  useEffect(() => {
    if (editingId === null) {
      return;
    }
    for (const row of displayRows) {
      if (row.kind === "group" && expandedRow !== row.key) {
        if (row.items.some((i) => i.id === editingId)) {
          setEditingId(null);
          break;
        }
      }
    }
  }, [expandedRow, displayRows, editingId, setEditingId]);
}
