import { plural } from "@/lib/fmt";
import { toast } from "@unkey/ui";
import { useState } from "react";
import type { EnvEntry } from "../env-file";
import {
  type RowErrors,
  type RowPatch,
  type SavedRow,
  type VariableRow,
  importVariableEntries,
  newVariableRow,
  pasteVariableEntriesAt,
} from "./variable-rows";

type SavedEdits = { values: ReadonlyMap<string, string>; removed: ReadonlySet<string> };

const NO_EDITS: SavedEdits = { values: new Map(), removed: new Set() };

export function useVariableRows(saved: readonly SavedRow[]) {
  const [fresh, setFresh] = useState<VariableRow[]>(() => [newVariableRow()]);
  const [errors, setErrors] = useState<Map<string, RowErrors>>(new Map());
  const [edits, setEdits] = useState<SavedEdits>(NO_EDITS);

  const kept = saved
    .filter((row) => !edits.removed.has(row.id))
    .map((row) => ({ ...row, value: edits.values.get(row.id) ?? row.value }));
  const rows: VariableRow[] = [...kept, ...fresh];

  const replaceFresh = (next: VariableRow[]) => {
    setFresh(next.filter((row) => row.origin.kind === "new"));
    setErrors(new Map());
  };

  const clearError = (id: string) =>
    setErrors((current) => {
      if (!current.has(id)) {
        return current;
      }
      const next = new Map(current);
      next.delete(id);
      return next;
    });

  const isSaved = (id: string) => saved.some((row) => row.id === id);

  return {
    rows,
    errors,
    removed: saved.filter((row) => edits.removed.has(row.id)),
    update: (id: string, patch: RowPatch) => {
      if (isSaved(id)) {
        const { value } = patch;
        if (value !== undefined) {
          setEdits((current) => ({ ...current, values: new Map(current.values).set(id, value) }));
        }
      } else {
        setFresh((current) => current.map((row) => (row.id === id ? { ...row, ...patch } : row)));
      }
      clearError(id);
    },
    remove: (id: string) => {
      if (isSaved(id)) {
        setEdits((current) => ({ ...current, removed: new Set(current.removed).add(id) }));
      } else {
        setFresh((current) => {
          const next = current.filter((row) => row.id !== id);
          return next.length > 0 ? next : [newVariableRow()];
        });
      }
      clearError(id);
    },
    add: (): string => {
      const row = newVariableRow();
      setFresh((current) => [...current, row]);
      return row.id;
    },
    importEntries: (entries: EnvEntry[]) => {
      const result = importVariableEntries(rows, entries);
      toastImportResult(result);
      if (result.added > 0) {
        replaceFresh(result.rows);
      }
    },
    pasteAt: (id: string, entries: EnvEntry[]) => {
      replaceFresh(pasteVariableEntriesAt(fresh, id, entries));
    },
    showErrors: setErrors,
    clearEdits: () => setEdits(NO_EDITS),
    reset: () => {
      replaceFresh([newVariableRow()]);
      setEdits(NO_EDITS);
    },
  };
}

export type VariableRows = ReturnType<typeof useVariableRows>;

function toastImportResult({ added, skipped }: { added: number; skipped: number }) {
  if (added === 0) {
    toast.info("All variables already exist");
  } else if (skipped > 0) {
    toast.success(`Imported ${plural(added, "variable")}, ${plural(skipped, "duplicate")} skipped`);
  } else {
    toast.success(`Imported ${plural(added, "variable")}`);
  }
}
