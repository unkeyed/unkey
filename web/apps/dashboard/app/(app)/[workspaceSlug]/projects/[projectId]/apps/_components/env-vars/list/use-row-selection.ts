import { useState } from "react";

export function useRowSelection<Row extends { id: string }>(rows: Row[]) {
  const [selectedIds, setSelectedIds] = useState<ReadonlySet<string>>(new Set());
  const [anchorId, setAnchorId] = useState<string | null>(null);
  const selected = rows.filter((row) => selectedIds.has(row.id));

  const clearSelection = () => {
    setSelectedIds(new Set());
    setAnchorId(null);
  };

  const toggleRowSelection = (id: string, shiftKey: boolean) => {
    const index = rows.findIndex((row) => row.id === id);
    const anchorIndex = shiftKey ? rows.findIndex((row) => row.id === anchorId) : -1;
    setAnchorId(id);
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (index !== -1 && anchorIndex !== -1) {
        for (const row of rows.slice(
          Math.min(anchorIndex, index),
          Math.max(anchorIndex, index) + 1,
        )) {
          next.add(row.id);
        }
      } else if (!next.delete(id)) {
        next.add(id);
      }
      return next;
    });
  };

  return { selectedIds, selected, toggleRowSelection, clearSelection };
}
