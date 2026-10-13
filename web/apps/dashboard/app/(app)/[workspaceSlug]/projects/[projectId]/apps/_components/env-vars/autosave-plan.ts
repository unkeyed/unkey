import type { EnvVar, EnvVarChange, VariableInput } from "@/lib/collections/deploy/env-vars";
import { ALL_ENVIRONMENTS, targetEnvironmentIds } from "../environment-scope";
import {
  type RowErrors,
  type VariableRow,
  isIncompleteRow,
  newVariableRow,
  splitVariableRows,
} from "./variables-table/variable-rows";

const UNKNOWN_AFTER_FAILURE = "unknown";

type EnvSnapshot = ReadonlyMap<string, string>;
export type ServerSnapshot = ReadonlyMap<string, EnvSnapshot>;

function entrySignature(variable: VariableInput): string {
  return JSON.stringify([variable.value, variable.kind]);
}

export function snapshotFromServer(envVars: readonly EnvVar[]): ServerSnapshot {
  const snapshot = new Map<string, Map<string, string>>();
  for (const v of envVars) {
    const env = snapshot.get(v.environmentId) ?? new Map<string, string>();
    env.set(v.key, entrySignature({ key: v.key, value: v.value, kind: v.type }));
    snapshot.set(v.environmentId, env);
  }
  return snapshot;
}

export function planChanges({
  server,
  environmentIds,
  targetIds,
  variables,
}: {
  server: ServerSnapshot;
  environmentIds: readonly string[];
  targetIds: readonly string[];
  variables: readonly VariableInput[];
}): EnvVarChange[] {
  return environmentIds.flatMap((environmentId) => {
    const current = server.get(environmentId) ?? new Map<string, string>();
    const wanted = targetIds.includes(environmentId) ? variables : [];
    const wantedKeys = new Set(wanted.map((v) => v.key));
    const set = wanted.filter((v) => current.get(v.key) !== entrySignature(v));
    const remove = [...current.keys()].filter((key) => !wantedKeys.has(key));
    return set.length > 0 || remove.length > 0 ? [{ environmentId, set, remove }] : [];
  });
}

export function applyChange(current: EnvSnapshot, change: EnvVarChange, ok: boolean): EnvSnapshot {
  const next = new Map(current);
  for (const key of change.remove) {
    if (ok) {
      next.delete(key);
    } else {
      next.set(key, UNKNOWN_AFTER_FAILURE);
    }
  }
  for (const v of change.set) {
    next.set(v.key, ok ? entrySignature(v) : UNKNOWN_AFTER_FAILURE);
  }
  return next;
}

export type HydratedTable =
  | { kind: "table"; rows: VariableRow[]; environmentId: string }
  | { kind: "unrepresentable" };

export function hydrateTable(
  envVars: readonly EnvVar[],
  environmentIds: readonly string[],
): HydratedTable {
  if (envVars.some((v) => v.type === "writeonly")) {
    return { kind: "unrepresentable" };
  }
  const values = new Map(envVars.map((v) => [v.key, v.value]));
  const rows = [...values].map(([key, value]) => newVariableRow({ key, value }));
  const usedEnvironments = [...new Set(envVars.map((v) => v.environmentId))];
  const environmentId =
    usedEnvironments.length === 1 && environmentIds.length > 1
      ? usedEnvironments[0]
      : ALL_ENVIRONMENTS;

  const { variables, errors } = splitVariableRows(rows);
  const exact =
    errors.size === 0 &&
    planChanges({
      server: snapshotFromServer(envVars),
      environmentIds,
      targetIds: targetEnvironmentIds(environmentId, environmentIds),
      variables,
    }).length === 0;
  return exact ? { kind: "table", rows, environmentId } : { kind: "unrepresentable" };
}

export type AutosaveStatus =
  | { kind: "loading" }
  | { kind: "load-failed" }
  | { kind: "unrepresentable" }
  | { kind: "failed" }
  | { kind: "invalid"; invalid: number }
  | { kind: "saving" }
  | { kind: "saved"; count: number };

export const STATUS_IS_DIRTY: Record<AutosaveStatus["kind"], boolean> = {
  loading: false,
  "load-failed": false,
  unrepresentable: false,
  failed: true,
  invalid: true,
  saving: true,
  saved: false,
};

export function resolveStatus({
  phase,
  failed,
  pending,
  invalid,
  count,
}: {
  phase: "loading" | "load-failed" | "unrepresentable" | "editing";
  failed: boolean;
  pending: boolean;
  invalid: number;
  count: number;
}): AutosaveStatus {
  if (phase !== "editing") {
    return { kind: phase };
  }
  if (failed) {
    return { kind: "failed" };
  }
  if (invalid > 0) {
    return { kind: "invalid", invalid };
  }
  if (pending) {
    return { kind: "saving" };
  }
  return { kind: "saved", count };
}

export function visibleRowErrors({
  errors,
  rows,
  settledRows,
  tableFocused,
}: {
  errors: ReadonlyMap<string, RowErrors>;
  rows: readonly VariableRow[];
  settledRows: readonly VariableRow[];
  tableFocused: boolean;
}): Map<string, RowErrors> {
  return new Map(
    [...errors].filter(([id]) => {
      const row = rows.find((r) => r.id === id);
      return (
        row !== undefined &&
        row === settledRows.find((r) => r.id === id) &&
        (!tableFocused || !isIncompleteRow(row))
      );
    }),
  );
}
