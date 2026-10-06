import type { VariableInput } from "@/lib/collections/deploy/env-vars";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";

export type VariableRow = { key: string; value: string; sensitive: boolean };

export type RowErrors = { key?: string; value?: string };

type ParsedRows =
  | { ok: true; variables: VariableInput[] }
  | { ok: false; errors: Map<number, RowErrors> };

export function parseVariableRows(rows: VariableRow[]): ParsedRows {
  const errors = new Map<number, RowErrors>();
  const variables: VariableInput[] = [];
  const seen = new Set<string>();

  rows.forEach((row, index) => {
    if (row.key.trim() === "" && row.value.trim() === "") {
      return;
    }
    const key = envVarKeySchema.safeParse(row.key);
    const value = envVarValueSchema.safeParse(row.value);
    const rowErrors: RowErrors = {
      key: key.success ? undefined : key.error.issues[0]?.message,
      value: value.success ? undefined : value.error.issues[0]?.message,
    };
    if (key.success && seen.has(key.data)) {
      rowErrors.key = "This name is already in the list. Use a different name.";
    }
    if (rowErrors.key || rowErrors.value) {
      errors.set(index, rowErrors);
      return;
    }
    if (key.success && value.success) {
      seen.add(key.data);
      variables.push({
        key: key.data,
        value: value.data,
        kind: row.sensitive ? "writeonly" : "recoverable",
      });
    }
  });

  return errors.size > 0 ? { ok: false, errors } : { ok: true, variables };
}

export function conflictErrors(
  rows: VariableRow[],
  existingKeys: ReadonlySet<string>,
): Map<number, RowErrors> {
  const errors = new Map<number, RowErrors>();
  rows.forEach((row, index) => {
    if (existingKeys.has(row.key.trim())) {
      errors.set(index, { key: "A variable with this name already exists. Use a different name." });
    }
  });
  return errors;
}
