import { catalogueFor, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  ACTION_LABELS,
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

export function supportedRowActions(row: PermissionRow): Action[] {
  return ACTIONS.filter((action) => row.actions?.[action] !== undefined);
}

export function rowActions(
  selection: PermissionSelection,
  rowId: string,
  row: PermissionRow,
): Action[];
export function rowActions(selection: PermissionSelection, rowId: string): Action[];
export function rowActions(
  selection: PermissionSelection,
  rowId: string,
  row?: PermissionRow,
): Action[] {
  const picked = selection[rowId];
  if (!picked) {
    return [];
  }
  const allowed = row === undefined ? ACTIONS : supportedRowActions(row);
  return allowed.filter((action) => picked.includes(action));
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
  row?: PermissionRow,
): PermissionSelection {
  const current =
    row === undefined ? rowActions(selection, rowId) : rowActions(selection, rowId, row);
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
        supportedRowActions(row).filter((action) => actions.includes(action)),
      ),
    selection,
  );
}

export function countSelectedActions(
  selection: PermissionSelection,
  rows: readonly PermissionRow[],
): number {
  return rows.reduce((total, row) => total + rowActions(selection, row.id, row).length, 0);
}

export function countSelectedRows(
  selection: PermissionSelection,
  rows: readonly PermissionRow[],
): number {
  return rows.filter((row) => rowActions(selection, row.id, row).length > 0).length;
}

export function selectInstances(current: readonly string[], next: readonly string[]): string[] {
  if (next.includes(ALL_INSTANCES) && !current.includes(ALL_INSTANCES)) {
    return [ALL_INSTANCES];
  }
  return next.filter((instance) => instance !== ALL_INSTANCES);
}

export function policyError(policy: Policy): string | null {
  const catalogue = catalogueFor(policy.scope);
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

export function actionLabel(actions: readonly Action[]): string {
  const ordered = ACTIONS.filter((action) => actions.includes(action));
  if (ordered.length === 0) {
    return "";
  }
  const labels = ordered.map((action) => ACTION_LABELS[action]);
  if (labels.length === 1) {
    return labels[0];
  }
  const head = labels.slice(0, -1).join(", ");
  return `${head} & ${labels[labels.length - 1]}`;
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
    const actions = rowActions(policy.selection, row.id, row);
    return actions.length === 0 ? [] : [`${row.label} ${actionLabel(actions)}`];
  });
  return { scopeLine, grants };
}

export function environmentLabel(appName: string | undefined, environmentName: string): string {
  return appName === undefined ? environmentName : `${appName} ${environmentName}`;
}

export function grantsPreview(
  grants: readonly string[],
  limit: number = GRANT_PREVIEW_LIMIT,
): { shown: string[]; more: number } {
  return { shown: grants.slice(0, limit), more: Math.max(grants.length - limit, 0) };
}
