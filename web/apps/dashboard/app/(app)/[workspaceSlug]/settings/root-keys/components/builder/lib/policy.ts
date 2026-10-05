import { CATALOGUES, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  ALL_INSTANCES,
  type Action,
  type PermissionRow,
  type PermissionSelection,
  type ResourceScope,
  rowOffers,
} from "./catalogue.types";

export type Policy = {
  scope: ResourceScope;
  instances: string[];
  selection: PermissionSelection;
};

export function newPolicy(scope: ResourceScope = "workspace"): Policy {
  return { scope, instances: [ALL_INSTANCES], selection: {} };
}

export function rowActions(selection: PermissionSelection, rowId: string): Action[] {
  const picked = selection[rowId];
  if (!picked) {
    return [];
  }
  return ACTIONS.filter((action) => picked.includes(action));
}

export function setRowActions(
  selection: PermissionSelection,
  rowId: string,
  actions: readonly Action[],
): PermissionSelection {
  const next = { ...selection };
  const kept = ACTIONS.filter((action) => actions.includes(action));
  if (kept.length === 0) {
    delete next[rowId];
    return next;
  }
  next[rowId] = kept;
  return next;
}

export function toggleRowAction(
  selection: PermissionSelection,
  rowId: string,
  action: Action,
  selected: boolean,
): PermissionSelection {
  const current = rowActions(selection, rowId);
  const next = selected ? [...current, action] : current.filter((entry) => entry !== action);
  return setRowActions(selection, rowId, next);
}

export function setRowsActions(
  selection: PermissionSelection,
  rows: readonly PermissionRow[],
  actions: readonly Action[],
): PermissionSelection {
  return rows.reduce<PermissionSelection>(
    (acc, row) =>
      setRowActions(
        acc,
        row.id,
        actions.filter((action) => rowOffers(row, action)),
      ),
    selection,
  );
}

export function countSelectedActions(
  selection: PermissionSelection,
  rows: readonly PermissionRow[],
): number {
  return rows.reduce((total, row) => total + rowActions(selection, row.id).length, 0);
}

export function selectInstances(current: readonly string[], next: readonly string[]): string[] {
  if (next.includes(ALL_INSTANCES) && !current.includes(ALL_INSTANCES)) {
    return [ALL_INSTANCES];
  }
  return next.filter((instance) => instance !== ALL_INSTANCES);
}

export function policyError(policy: Policy): string | null {
  const catalogue = CATALOGUES[policy.scope];
  if (catalogue.instanceNoun !== null && policy.instances.length === 0) {
    return `Select one or more ${catalogue.instanceNoun}`;
  }
  if (
    policy.instances.length === 0 ||
    countSelectedActions(policy.selection, catalogueRows(catalogue)) === 0
  ) {
    return "At least one permission required";
  }
  return null;
}

export function isPolicyComplete(policy: Policy): boolean {
  return policyError(policy) === null;
}
