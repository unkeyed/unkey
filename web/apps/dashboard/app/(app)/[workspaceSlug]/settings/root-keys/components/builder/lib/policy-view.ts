import { CATALOGUES, catalogueRows } from "./catalogue";
import { ACTIONS, ALL_INSTANCES, type Action } from "./catalogue.types";
import { type Policy, rowActions } from "./policy";

export type PolicySummary = {
  scopeLine: string;
  grants: string[];
};

export function actionLabel(actions: readonly Action[]): string {
  const ordered = ACTIONS.filter((action) => actions.includes(action));
  if (ordered.length === 0) {
    return "";
  }
  const capitalised = ordered.map(
    (action) => `${action.charAt(0).toUpperCase()}${action.slice(1)}`,
  );
  if (capitalised.length === 1) {
    return capitalised[0];
  }
  const head = capitalised.slice(0, -1).join(", ");
  return `${head} & ${capitalised[capitalised.length - 1]}`;
}

export function policySummary(
  policy: Policy,
  instanceLabels: Record<string, string> = {},
): PolicySummary {
  const catalogue = CATALOGUES[policy.scope];
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

export function environmentLabel(appName: string | undefined, environmentName: string): string {
  return appName === undefined ? environmentName : `${appName} ${environmentName}`;
}
