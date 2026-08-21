import { catalogueFor, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  type Action,
  type PermissionRow,
  type PermissionSelection,
  type ResourceScope,
} from "./catalogue.types";

export const ALL_INSTANCES = "__all__";

export const GRANT_PREVIEW_LIMIT = 3;

export type Policy = {
  scope: ResourceScope;
  instances: string[];
  selection: PermissionSelection;
};

export type PolicySummary = {
  scopeLine: string;
  grants: string[];
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

export function isRowActionSelected(
  selection: PermissionSelection,
  rowId: string,
  action: Action,
): boolean {
  return rowActions(selection, rowId).includes(action);
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
    (acc, row) => setRowActions(acc, row.id, actions),
    selection,
  );
}

export function countSelectedActions(
  selection: PermissionSelection,
  rows: readonly PermissionRow[],
): number {
  return rows.reduce((total, row) => total + rowActions(selection, row.id).length, 0);
}

export function countSelectedRows(
  selection: PermissionSelection,
  rows: readonly PermissionRow[],
): number {
  return rows.filter((row) => rowActions(selection, row.id).length > 0).length;
}

export function isPolicyComplete(policy: Policy): boolean {
  if (policy.instances.length === 0) {
    return false;
  }
  return countSelectedActions(policy.selection, catalogueRows(catalogueFor(policy.scope))) > 0;
}

export function actionLabel(actions: readonly Action[]): string {
  const ordered = ACTIONS.filter((action) => actions.includes(action));
  if (ordered.length === 0) {
    return "";
  }
  const capitalised = `${ordered[0].charAt(0).toUpperCase()}${ordered[0].slice(1)}`;
  if (ordered.length === 1) {
    return capitalised;
  }
  const head = [capitalised, ...ordered.slice(1, -1)].join(", ");
  return `${head} & ${ordered[ordered.length - 1]}`;
}

export function policySummary(
  policy: Policy,
  instanceLabels: Record<string, string> = {},
): PolicySummary {
  const catalogue = catalogueFor(policy.scope);
  const named = policy.instances.filter((instance) => instance !== ALL_INSTANCES);
  const scopeLine =
    named.length === 0
      ? catalogue.allLabel
      : named.map((instance) => instanceLabels[instance] ?? instance).join(", ");
  const grants = catalogueRows(catalogue).flatMap((row) => {
    const actions = rowActions(policy.selection, row.id);
    return actions.length === 0 ? [] : [`${row.label} ${actionLabel(actions)}`];
  });
  return { scopeLine, grants };
}

export function grantsPreview(
  grants: readonly string[],
  limit: number = GRANT_PREVIEW_LIMIT,
): { shown: string[]; more: number } {
  return { shown: grants.slice(0, limit), more: Math.max(grants.length - limit, 0) };
}
