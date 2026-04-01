import { collection } from "@/lib/collections";
import { useCallback, useEffect, useRef, useState } from "react";
import type { DisplayRow } from "./env-var-item-row";

function getRowIds(row: DisplayRow): string[] {
  return row.kind === "single" ? [row.item.id] : row.items.map((i) => i.id);
}

export function useRowSelection(displayRows: DisplayRow[]) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [isDeleting, setIsDeleting] = useState(false);
  const lastClickedIndexRef = useRef<number | null>(null);

  // Prune selected IDs that no longer exist in display rows
  useEffect(() => {
    setSelectedIds((prev) => {
      if (prev.size === 0) {
        return prev;
      }
      const visibleIds = new Set<string>();
      for (const row of displayRows) {
        if (row.kind === "single") {
          visibleIds.add(row.item.id);
        } else {
          for (const item of row.items) {
            visibleIds.add(item.id);
          }
        }
      }
      let changed = false;
      const next = new Set<string>();
      for (const id of prev) {
        if (visibleIds.has(id)) {
          next.add(id);
        } else {
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [displayRows]);

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
        } else if (row.kind === "single") {
          const id = row.item.id;
          if (next.has(id)) {
            next.delete(id);
          } else {
            next.add(id);
          }
        } else {
          const ids = getRowIds(row);
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

        lastClickedIndexRef.current = rowIndex;
        return next;
      });
    },
    [displayRows],
  );

  const isRowSelected = useCallback(
    (row: DisplayRow): boolean | "partial" => {
      if (row.kind === "single") {
        return selectedIds.has(row.item.id);
      }
      let selectedCount = 0;
      for (const item of row.items) {
        if (selectedIds.has(item.id)) {
          selectedCount++;
        }
      }
      if (selectedCount === 0) {
        return false;
      }
      if (selectedCount === row.items.length) {
        return true;
      }
      return "partial";
    },
    [selectedIds],
  );

  const handleBulkDelete = useCallback(() => {
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

  return {
    selectedIds,
    isDeleting,
    toggleRowSelection,
    isRowSelected,
    handleBulkDelete,
    clearSelection,
  };
}
