"use client";

import { collection } from "@/lib/collections";
import type { Environment } from "@/lib/collections/deploy/environments";
import { cn } from "@/lib/utils";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Badge, Checkbox } from "@unkey/ui";
import { useCallback, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import {
  type DisplayRow,
  type EnvVarItem,
  EnvVarItemRow,
  TimestampBadge,
  groupByKey,
  rowKey,
  rowTime,
} from "./env-var-item-row";
import { EnvVarSelectionBar } from "./env-var-selection-bar";
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
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [isDeleting, setIsDeleting] = useState(false);
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

  // Prune selected IDs that no longer exist in display rows
  const allVisibleIds = useMemo(() => {
    const ids = new Set<string>();
    for (const row of displayRows) {
      if (row.kind === "single") {
        ids.add(row.item.id);
      } else {
        for (const item of row.items) {
          ids.add(item.id);
        }
      }
    }
    return ids;
  }, [displayRows]);

  useEffect(() => {
    setSelectedIds((prev) => {
      let changed = false;
      const next = new Set<string>();
      for (const id of prev) {
        if (allVisibleIds.has(id)) {
          next.add(id);
        } else {
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [allVisibleIds]);

  const lastClickedIndexRef = useRef<number | null>(null);

  const getRowIds = useCallback((row: DisplayRow): string[] => {
    return row.kind === "single" ? [row.item.id] : row.items.map((i) => i.id);
  }, []);

  const toggleRowSelection = useCallback(
    (rowIndex: number, shiftKey: boolean) => {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        const row = displayRows[rowIndex];
        if (!row) {
          return prev;
        }

        if (shiftKey && lastClickedIndexRef.current !== null) {
          const start = Math.min(lastClickedIndexRef.current, rowIndex);
          const end = Math.max(lastClickedIndexRef.current, rowIndex);
          for (let i = start; i <= end; i++) {
            const r = displayRows[i];
            if (r) {
              for (const id of getRowIds(r)) {
                next.add(id);
              }
            }
          }
        } else {
          const ids = getRowIds(row);
          if (row.kind === "single") {
            if (next.has(ids[0])) {
              next.delete(ids[0]);
            } else {
              next.add(ids[0]);
            }
          } else {
            const allSelected = ids.every((id) => next.has(id));
            if (allSelected) {
              for (const id of ids) {
                next.delete(id);
              }
            } else {
              for (const id of ids) {
                next.add(id);
              }
            }
          }
        }

        lastClickedIndexRef.current = rowIndex;
        return next;
      });
    },
    [displayRows, getRowIds],
  );

  const isRowSelected = useCallback(
    (row: DisplayRow): boolean | "partial" => {
      if (row.kind === "single") {
        return selectedIds.has(row.item.id);
      }
      const groupIds = row.items.map((i) => i.id);
      const selectedCount = groupIds.filter((id) => selectedIds.has(id)).length;
      if (selectedCount === 0) {
        return false;
      }
      if (selectedCount === groupIds.length) {
        return true;
      }
      return "partial";
    },
    [selectedIds],
  );

  const handleBulkDelete = useCallback(async () => {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) {
      return;
    }
    setIsDeleting(true);
    try {
      collection.envVars.delete(ids);
      setSelectedIds(new Set());
    } finally {
      setIsDeleting(false);
    }
  }, [selectedIds]);

  const clearSelection = useCallback(() => setSelectedIds(new Set()), []);

  useCloseEditOnGroupCollapse(displayRows, expandedGroups, editingId, setEditingId);

  if (isLoading) {
    return <EnvVarsSkeleton />;
  }

  if (displayRows.length === 0) {
    return <EnvVarsEmpty searchQuery={searchQuery} />;
  }

  return (
    <>
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
        {displayRows.map((row, index) => {
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
                isSelected={selectedIds.has(item.id)}
                onToggleSelection={(shiftKey) => toggleRowSelection(index, shiftKey)}
                hasSelection={selectedIds.size > 0}
              />
            );
          }

          // Group row
          const isExpanded = expandedGroups.has(row.key);
          const selected = isRowSelected(row);
          return (
            <GroupRow
              key={`group-${row.key}`}
              row={row}
              isExpanded={isExpanded}
              selected={selected}
              deferredQuery={deferredQuery}
              editingId={editingId}
              onToggleGroup={() => toggleGroup(row.key)}
              onToggleSelection={(shiftKey) => toggleRowSelection(index, shiftKey)}
              onEdit={setEditingId}
              onCloseEdit={closeEdit}
              hasSelection={selectedIds.size > 0}
            />
          );
        })}
      </div>
      <EnvVarSelectionBar
        selectedCount={selectedIds.size}
        onDelete={handleBulkDelete}
        onClearSelection={clearSelection}
        isDeleting={isDeleting}
      />
    </>
  );
}

type GroupRowProps = {
  row: DisplayRow & { kind: "group" };
  isExpanded: boolean;
  selected: boolean | "partial";
  deferredQuery: string;
  editingId: string | null;
  onToggleGroup: () => void;
  onToggleSelection: (shiftKey: boolean) => void;
  onEdit: (id: string) => void;
  onCloseEdit: () => void;
  hasSelection: boolean;
};

function GroupRow({
  row,
  isExpanded,
  selected,
  deferredQuery,
  editingId,
  onToggleGroup,
  onToggleSelection,
  onEdit,
  onCloseEdit,
  hasSelection,
}: GroupRowProps) {
  const isChecked = selected === true;
  const isIndeterminate = selected === "partial";

  return (
    <div>
      <div
        //biome-ignore lint/a11y/useSemanticElements: its okay
        role="button"
        tabIndex={0}
        onClick={onToggleGroup}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onToggleGroup();
          }
        }}
        className="group flex items-center hover:bg-grayA-2 transition-colors cursor-pointer"
      >
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: checkbox handles keyboard interaction */}
        <div
          className="pl-4 flex items-center w-8 shrink-0"
          onClick={(e) => {
            e.stopPropagation();
            onToggleSelection(e.shiftKey);
          }}
        >
          <Checkbox
            checked={isIndeterminate ? "indeterminate" : isChecked}
            className={cn(
              "size-4 [&_svg]:size-3",
              isChecked || isIndeterminate || hasSelection
                ? "opacity-100"
                : "opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto focus-visible:opacity-100 focus-visible:pointer-events-auto",
            )}
            onCheckedChange={() => { }}
          />
        </div>
        <div className="flex-4 min-w-0 py-3.5 flex items-center">
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
              <div className="text-[13px] mt-1 text-gray-11 capitalize">All Environments</div>
            </div>
          </div>
        </div>
        <div className="flex-4 min-w-0 py-3.5 flex items-center pr-3">
          <span className="text-[13px] text-gray-11 transition-colors pl-2">
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
              onEdit={() => onEdit(item.id)}
              onCloseEdit={onCloseEdit}
              selectable={false}
            />
          ))}
        </div>
      )}
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
