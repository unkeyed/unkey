import { toast } from "@unkey/ui";
import { useState } from "react";
import type { EnvEntry } from "../../env-file";
import {
  type RowErrors,
  type VariableRow,
  importVariableEntries,
  newVariableRow,
  pasteVariableEntriesAt,
} from "./variable-rows";

export function useVariableRows() {
  const [rows, setRows] = useState<VariableRow[]>(() => [newVariableRow()]);
  const [errors, setErrors] = useState<Map<string, RowErrors>>(new Map());

  const replaceRows = (next: VariableRow[]) => {
    setRows(next);
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

  return {
    rows,
    errors,
    update: (id: string, patch: Partial<Omit<VariableRow, "id">>) => {
      setRows((current) => current.map((row) => (row.id === id ? { ...row, ...patch } : row)));
      clearError(id);
    },
    remove: (id: string) => {
      setRows((current) => {
        const next = current.filter((row) => row.id !== id);
        return next.length > 0 ? next : [newVariableRow()];
      });
      clearError(id);
    },
    add: (): string => {
      const row = newVariableRow();
      setRows((current) => [...current, row]);
      return row.id;
    },
    importEntries: (entries: EnvEntry[]) => {
      const result = importVariableEntries(rows, entries);
      toastImportResult(result);
      if (result.added > 0) {
        replaceRows(result.rows);
      }
    },
    pasteAt: (id: string, entries: EnvEntry[]) => {
      replaceRows(pasteVariableEntriesAt(rows, id, entries));
    },
    showErrors: setErrors,
    reset: () => replaceRows([newVariableRow()]),
  };
}

export type VariableRows = ReturnType<typeof useVariableRows>;

function toastImportResult({ added, skipped }: { added: number; skipped: number }) {
  if (added === 0) {
    toast.info("All variables already exist");
  } else if (skipped > 0) {
    toast.success(
      `Imported ${variables(added)}, ${skipped} ${skipped === 1 ? "duplicate" : "duplicates"} skipped`,
    );
  } else {
    toast.success(`Imported ${variables(added)}`);
  }
}

function variables(count: number): string {
  return `${count} ${count === 1 ? "variable" : "variables"}`;
}
