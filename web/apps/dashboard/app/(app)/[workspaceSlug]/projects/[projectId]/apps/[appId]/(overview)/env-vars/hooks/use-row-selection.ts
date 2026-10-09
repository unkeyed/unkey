import { collection } from "@/lib/collections";
import {
  type EnvVar,
  envVarErrorToast,
  makeVariablesSensitive,
} from "@/lib/collections/deploy/env-vars";
import { plural } from "@/lib/fmt";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useState } from "react";

export function useRowSelection(rows: EnvVar[], selectionResetKey: string) {
  const [selectedIds, setSelectedIds] = useState<ReadonlySet<string>>(new Set());
  const [anchorId, setAnchorId] = useState<string | null>(null);
  const [appliedResetKey, setAppliedResetKey] = useState(selectionResetKey);
  if (appliedResetKey !== selectionResetKey) {
    setAppliedResetKey(selectionResetKey);
    setSelectedIds(new Set());
    setAnchorId(null);
  }
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

  const handleBulkDelete = () => {
    if (selected.length > 0) {
      collection.envVars.delete(selected.map((item) => item.id));
    }
    clearSelection();
  };

  const makeSensitive = useMutation({
    mutationFn: makeVariablesSensitive,
    onSuccess: (updated) => {
      toast.success(`Marked ${plural(updated, "variable")} as sensitive`);
    },
    onError: (err) => {
      const { message, description } = envVarErrorToast(
        err,
        "Failed to mark variables as sensitive",
      );
      toast.error(message, { description });
    },
    onSettled: clearSelection,
  });

  const handleBulkMakeSensitive = () => {
    if (makeSensitive.isLoading) {
      return;
    }
    const recoverable = selected.filter((item) => item.type === "recoverable");
    if (recoverable.length === 0) {
      toast.info("No recoverable variables are selected");
      clearSelection();
      return;
    }
    makeSensitive.mutate(recoverable);
  };

  return {
    selectedIds,
    selectedCount: selected.length,
    toggleRowSelection,
    handleBulkDelete,
    handleBulkMakeSensitive,
    isMakingSensitive: makeSensitive.isLoading,
    clearSelection,
  };
}
