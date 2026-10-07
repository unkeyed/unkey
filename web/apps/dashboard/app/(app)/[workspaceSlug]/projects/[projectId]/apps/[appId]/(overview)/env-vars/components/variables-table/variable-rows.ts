import type { VariableInput } from "@/lib/collections/deploy/env-vars";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import type { EnvEntry } from "../../env-file";

const DUPLICATE_NAME = "This key appears more than once. Rename or remove the extra rows.";

export type VariableRow = { id: string; key: string; value: string; sensitive: boolean };

export type RowErrors = { key?: string; value?: string };

type ParsedRows =
  | { ok: true; variables: VariableInput[] }
  | { ok: false; errors: Map<string, RowErrors> };

export function newVariableRow(entry: EnvEntry = { key: "", value: "" }): VariableRow {
  return { id: crypto.randomUUID(), key: entry.key, value: entry.value, sensitive: false };
}

export function parseVariableRows(rows: VariableRow[]): ParsedRows {
  const errors = new Map<string, RowErrors>();
  const variables: VariableInput[] = [];
  const seen = new Set<string>();

  for (const row of rows) {
    if (row.key.trim() === "" && row.value.trim() === "") {
      continue;
    }
    const key = envVarKeySchema.safeParse(row.key);
    const value = envVarValueSchema.safeParse(row.value);
    const rowErrors: RowErrors = {
      key: key.success ? undefined : key.error.issues[0]?.message,
      value: value.success ? undefined : value.error.issues[0]?.message,
    };
    if (key.success && seen.has(key.data)) {
      rowErrors.key = DUPLICATE_NAME;
    }
    if (rowErrors.key || rowErrors.value) {
      errors.set(row.id, rowErrors);
      continue;
    }
    if (key.success && value.success) {
      seen.add(key.data);
      variables.push({
        key: key.data,
        value: value.data,
        kind: row.sensitive ? "writeonly" : "recoverable",
      });
    }
  }

  return errors.size > 0 ? { ok: false, errors } : { ok: true, variables };
}

const TAKEN_NAME = "A variable with this key already exists.";

export function conflictErrors(
  rows: VariableRow[],
  existingKeys: ReadonlySet<string>,
): Map<string, RowErrors> {
  return new Map(
    rows
      .filter((row) => existingKeys.has(row.key.trim()))
      .map((row) => [row.id, { key: TAKEN_NAME }]),
  );
}

export function importVariableEntries(
  rows: readonly VariableRow[],
  entries: readonly EnvEntry[],
): { rows: VariableRow[]; added: number; skipped: number } {
  // Last occurrence wins, as in a .env file.
  const deduped = new Map(entries.map((entry) => [entry.key, entry]));
  const existingKeys = new Set(rows.filter((row) => row.key !== "").map((row) => row.key));
  const added = [...deduped.values()].filter((entry) => !existingKeys.has(entry.key));
  const skipped = deduped.size - added.length;

  if (added.length === 0) {
    return { rows: [...rows], added: 0, skipped };
  }

  return {
    rows: [
      ...rows.filter((row) => row.key !== "" || row.value !== ""),
      ...added.map(newVariableRow),
    ],
    added: added.length,
    skipped,
  };
}

export function pasteVariableEntriesAt(
  rows: readonly VariableRow[],
  rowId: string,
  entries: readonly EnvEntry[],
): VariableRow[] {
  if (entries.length === 0) {
    return [...rows];
  }
  const [first, ...rest] = entries;
  return [
    ...rows.map((row) => (row.id === rowId ? { ...row, key: first.key, value: first.value } : row)),
    ...rest.map(newVariableRow),
  ];
}
