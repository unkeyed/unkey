import { collection } from "@/lib/collections";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useCallback, useRef, useState } from "react";
import type { DisplayRow } from "../components/list/env-var-item-row";

function getRowIds(row: DisplayRow): string[] {
  return row.kind === "single" ? [row.item.id] : row.items.map((i) => i.id);
}

export function useRowSelection(displayRows: DisplayRow[], envVars: EnvVar[] | undefined) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const lastClickedIndexRef = useRef<number | null>(null);

  // Resolve against every row of the app, not the filtered `displayRows`: the
  // search and environment filters hide rows the user selected before
  // filtering. A row id is derived from the key, so a rename since the
  // selection was made also leaves ids that match nothing.
  const resolveSelection = useCallback(
    () => (envVars ?? []).filter((item) => selectedIds.has(item.id)),
    [envVars, selectedIds],
  );

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
    const ids = resolveSelection().map((item) => item.id);
    if (ids.length === 0) {
      setSelectedIds(new Set());
      return;
    }
    collection.envVars.delete(ids);
    setSelectedIds(new Set());
  }, [resolveSelection]);

  const toggleItemSelection = useCallback((itemId: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(itemId)) {
        next.delete(itemId);
      } else {
        next.add(itemId);
      }
      return next;
    });
  }, []);

  const makeSensitiveMutation = trpc.deploy.envVar.makeSensitive.useMutation();

  const handleBulkMakeSensitive = useCallback(async () => {
    const recoverable = resolveSelection().filter((item) => item.type === "recoverable");
    if (recoverable.length === 0) {
      toast.info("No recoverable variables are selected");
      setSelectedIds(new Set());
      return;
    }
    try {
      const { updated } = await makeSensitiveMutation.mutateAsync({
        appId: recoverable[0].appId,
        targets: recoverable.map((v) => ({ environmentId: v.environmentId, key: v.key })),
      });
      toast.success(`Marked ${updated} variable${updated === 1 ? "" : "s"} as sensitive`);
    } catch {
      toast.error("Failed to mark variables as sensitive");
    }
    await collection.envVars.utils.refetch().catch(() => {});
    setSelectedIds(new Set());
  }, [resolveSelection, makeSensitiveMutation.mutateAsync]);

  const clearSelection = useCallback(() => setSelectedIds(new Set()), []);

  return {
    selectedIds,
    toggleRowSelection,
    toggleItemSelection,
    isRowSelected,
    handleBulkDelete,
    handleBulkMakeSensitive,
    clearSelection,
  };
}
