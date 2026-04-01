import { collection } from "@/lib/collections";
import { useCallback, useRef, useState } from "react";
import type { DisplayRow } from "../components/list/env-var-item-row";

function getRowIds(row: DisplayRow): string[] {
  return row.kind === "single" ? [row.item.id] : row.items.map((i) => i.id);
}

export function useRowSelection(displayRows: DisplayRow[]) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const lastClickedIndexRef = useRef<number | null>(null);

  const toggleRowSelection = useCallback(
    (rowIndex: number, shiftKey: boolean) => {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        const row = displayRows[rowIndex];
        if (!row) {
          return prev;
        }

        // Shift+click: select entire range from last clicked row
        // Normal click: toggle row, deselect if all IDs selected, select otherwise
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
          const allSelected = ids.every((id) => next.has(id));
          for (const id of ids) {
            if (allSelected) {
              next.delete(id);
            } else {
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
      const ids = getRowIds(row);
      const selectedCount = ids.filter((id) => selectedIds.has(id)).length;
      if (selectedCount === 0) {
        return false;
      }
      return selectedCount === ids.length ? true : "partial";
    },
    [selectedIds],
  );

  const handleBulkDelete = useCallback(() => {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) {
      return;
    }
    collection.envVars.delete(ids);
    setSelectedIds(new Set());
  }, [selectedIds]);

  const clearSelection = useCallback(() => setSelectedIds(new Set()), []);

  return {
    selectedIds,
    toggleRowSelection,
    isRowSelected,
    handleBulkDelete,
    clearSelection,
  };
}
