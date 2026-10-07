import { collection } from "@/lib/collections";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useCallback, useRef, useState } from "react";

export function useRowSelection(rows: EnvVar[], envVars: EnvVar[] | undefined) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const lastClickedIndexRef = useRef<number | null>(null);

  const resolveSelection = useCallback(
    () => (envVars ?? []).filter((item) => selectedIds.has(item.id)),
    [envVars, selectedIds],
  );

  const toggleRowSelection = useCallback(
    (rowIndex: number, shiftKey: boolean) => {
      setSelectedIds((prev) => {
        const row = rows[rowIndex];
        if (!row) {
          return prev;
        }
        const next = new Set(prev);
        if (shiftKey && lastClickedIndexRef.current !== null) {
          const start = Math.min(lastClickedIndexRef.current, rowIndex);
          const end = Math.max(lastClickedIndexRef.current, rowIndex);
          for (const r of rows.slice(start, end + 1)) {
            next.add(r.id);
          }
        } else if (next.has(row.id)) {
          next.delete(row.id);
        } else {
          next.add(row.id);
        }
        lastClickedIndexRef.current = rowIndex;
        return next;
      });
    },
    [rows],
  );

  const handleBulkDelete = useCallback(() => {
    const ids = resolveSelection().map((item) => item.id);
    if (ids.length > 0) {
      collection.envVars.delete(ids);
    }
    setSelectedIds(new Set());
  }, [resolveSelection]);

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
    handleBulkDelete,
    handleBulkMakeSensitive,
    clearSelection,
  };
}
