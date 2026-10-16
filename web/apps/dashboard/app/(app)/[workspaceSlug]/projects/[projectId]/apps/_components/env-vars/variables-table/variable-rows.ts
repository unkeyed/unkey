import type { EnvVar, VariableInput } from "@/lib/collections/deploy/env-vars";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import type { EnvEntry } from "../env-file";

const DUPLICATE_NAME = "This key appears more than once. Rename or remove the extra rows.";

const TAKEN_NAME = "A variable with this key already exists.";

type RowLock = "none" | "sensitive" | "varies";

type SavedOrigin = {
  kind: "saved";
  lock: RowLock;
  initial: string;
  envVars: readonly EnvVar[];
};

type RowFields = { id: string; key: string; value: string; sensitive: boolean };

export type SavedRow = RowFields & { origin: SavedOrigin };

export type VariableRow = RowFields & { origin: { kind: "new" } | SavedOrigin };

export type RowPatch = Partial<Pick<VariableRow, "key" | "value" | "sensitive">>;

export type RowView = "new" | RowLock;

export type RowErrors = { key?: string; value?: string };

export function newVariableRow(entry: EnvEntry = { key: "", value: "" }): VariableRow {
  return {
    id: crypto.randomUUID(),
    key: entry.key,
    value: entry.value,
    sensitive: false,
    origin: { kind: "new" },
  };
}

export function rowView(row: VariableRow): RowView {
  return row.origin.kind === "new" ? "new" : row.origin.lock;
}

export function isBlankRow(row: VariableRow): boolean {
  return row.key.trim() === "" && row.value.trim() === "";
}

export function isChangedRow(row: VariableRow): boolean {
  return row.origin.kind === "new" ? !isBlankRow(row) : row.value !== row.origin.initial;
}

export function splitVariableRows(rows: readonly VariableRow[]): {
  variables: VariableInput[];
  errors: Map<string, RowErrors>;
} {
  const errors = new Map<string, RowErrors>();
  const variables: VariableInput[] = [];
  const seen = new Map<string, string>();

  for (const row of rows) {
    if (row.origin.kind === "saved") {
      seen.set(row.key, TAKEN_NAME);
      const value = isChangedRow(row) ? envVarValueSchema.safeParse(row.value) : null;
      if (value && !value.success) {
        errors.set(row.id, { value: value.error.issues[0]?.message });
      }
      continue;
    }
    if (isBlankRow(row)) {
      continue;
    }
    const key = envVarKeySchema.safeParse(row.key);
    const value = envVarValueSchema.safeParse(row.value);
    const rowErrors: RowErrors = {
      key: key.success ? seen.get(key.data) : key.error.issues[0]?.message,
      value: value.success ? undefined : value.error.issues[0]?.message,
    };
    if (rowErrors.key || rowErrors.value) {
      errors.set(row.id, rowErrors);
      continue;
    }
    if (key.success && value.success) {
      seen.set(key.data, DUPLICATE_NAME);
      variables.push({
        key: key.data,
        value: value.data,
        kind: row.sensitive ? "writeonly" : "recoverable",
      });
    }
  }

  return { variables, errors };
}

export function conflictErrors(
  rows: readonly VariableRow[],
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
  const deduped = new Map(entries.map((entry) => [entry.key, entry]));
  const existingKeys = new Set(rows.map((row) => row.key.trim()).filter((key) => key !== ""));
  const added = [...deduped.values()].filter((entry) => !existingKeys.has(entry.key));
  const skipped = deduped.size - added.length;

  if (added.length === 0) {
    return { rows: [...rows], added: 0, skipped };
  }

  return {
    rows: [...rows.filter((row) => !isBlankRow(row)), ...added.map(newVariableRow)],
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

function savedLock(variables: readonly EnvVar[]): RowLock {
  if (variables.some((v) => v.type === "writeonly")) {
    return "sensitive";
  }
  return new Set(variables.map((v) => v.value)).size === 1 ? "none" : "varies";
}

export function savedVariableRows(
  envVars: readonly EnvVar[],
  targetIds: readonly string[],
): SavedRow[] {
  const targets = new Set(targetIds);
  const byKey = new Map<string, EnvVar[]>();
  for (const variable of envVars) {
    if (targets.has(variable.environmentId)) {
      byKey.set(variable.key, [...(byKey.get(variable.key) ?? []), variable]);
    }
  }
  return [...byKey]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, variables]) => {
      const lock = savedLock(variables);
      const initial = lock === "none" ? variables[0].value : "";
      return {
        id: `saved:${key}`,
        key,
        value: initial,
        sensitive: lock === "sensitive",
        origin: { kind: "saved", lock, initial, envVars: variables },
      };
    });
}

export function savedVariableChanges(
  rows: readonly VariableRow[],
  removed: readonly SavedRow[],
): { edited: EnvVar[]; removed: EnvVar[] } {
  return {
    edited: rows.flatMap((row) =>
      row.origin.kind === "saved" && isChangedRow(row)
        ? row.origin.envVars.map((variable) => ({ ...variable, value: row.value }))
        : [],
    ),
    removed: removed.flatMap((row) => row.origin.envVars),
  };
}
