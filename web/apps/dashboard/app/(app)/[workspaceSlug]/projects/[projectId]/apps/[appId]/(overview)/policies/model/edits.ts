import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import { type PolicyRow, toPolicyInput } from "@/lib/collections/deploy/policies";
import {
  POLICY_ERRORS,
  POLICY_LIMITS,
  type PolicyInput,
  policyMatchKey,
} from "@/lib/collections/deploy/policies.schema";
import { duplicateNameError, otherMatchKeys } from "./duplicate-name";
import { type Env, type MergedPolicy, mergePolicies, policyInEnv } from "./merge";
import { movePolicy } from "./reorder";

export type PolicyLists = Record<Env, PolicyRow[]>;

export type PolicyListsScope = {
  projectId: string;
  appId: string;
  environments: Record<Env, string>;
};

export type PolicyPatch = PolicyInput | { enabled: boolean } | { _order: number };

export type PolicyOp =
  | { type: "insert"; row: PolicyRow }
  | { type: "update"; row: PolicyRow; patch: PolicyPatch }
  | { type: "delete"; row: PolicyRow };

export type PolicyEditResult =
  | { type: "apply"; ops: PolicyOp[] }
  | { type: "reject"; message: string };

export type PolicyEdit = (lists: PolicyLists, scope: PolicyListsScope) => PolicyEditResult;

export function replacesList(ops: readonly PolicyOp[]): boolean {
  return ops.some((op) => op.type !== "update" || "_order" in op.patch);
}

const reject = (message: string): PolicyEditResult => ({ type: "reject", message });
const apply = (ops: PolicyOp[]): PolicyEditResult => ({ type: "apply", ops });

const merge = (lists: PolicyLists): MergedPolicy[] =>
  mergePolicies(lists.production, lists.preview);

const copiesOf = (merged: MergedPolicy[], key: string): PolicyRow[] =>
  ENVIRONMENT_KINDS.flatMap((env) => policyInEnv(merged, key, env) ?? []);

const atCapacity = (lists: PolicyLists, envs: readonly Env[]) =>
  envs.some((env) => lists[env].length >= POLICY_LIMITS.maxPolicies);

function newRow(
  policy: PolicyInput,
  env: Env,
  lists: PolicyLists,
  scope: PolicyListsScope,
): PolicyRow {
  return {
    ...policy,
    id: `new:${policyMatchKey(policy.type, policy.name)}`,
    environmentId: scope.environments[env],
    projectId: scope.projectId,
    appId: scope.appId,
    _order: (lists[env].at(-1)?._order ?? -1) + 1,
  };
}

export function insertPolicy(policy: PolicyInput, targets: readonly Env[]): PolicyEdit {
  return (lists, scope) => {
    if (atCapacity(lists, targets)) {
      return reject(POLICY_ERRORS.atCapacity);
    }
    const duplicate = duplicateNameError(policy, otherMatchKeys(merge(lists), null));
    if (duplicate) {
      return reject(duplicate);
    }
    return apply(
      targets.map((env) => ({ type: "insert", row: newRow(policy, env, lists, scope) })),
    );
  };
}

export function setEnabled(key: string, env: Env, enabled: boolean): PolicyEdit {
  return (lists, scope) => {
    const merged = merge(lists);
    const copy = policyInEnv(merged, key, env);
    if (copy) {
      return apply(
        copy.enabled === enabled ? [] : [{ type: "update", row: copy, patch: { enabled } }],
      );
    }
    const [source] = copiesOf(merged, key);
    if (!source) {
      return reject(POLICY_ERRORS.gone);
    }
    if (!enabled) {
      return apply([]);
    }
    if (atCapacity(lists, [env])) {
      return reject(POLICY_ERRORS.atCapacity);
    }
    if (duplicateNameError(source, otherMatchKeys(merged, key))) {
      return reject(POLICY_ERRORS.sharedName);
    }
    const policy = { ...toPolicyInput(source), enabled };
    return apply([{ type: "insert", row: newRow(policy, env, lists, scope) }]);
  };
}

export function updatePolicy(key: string, policy: PolicyInput): PolicyEdit {
  return (lists) => {
    const merged = merge(lists);
    const copies = copiesOf(merged, key);
    if (copies.length === 0) {
      return reject(POLICY_ERRORS.gone);
    }
    const duplicate = duplicateNameError(policy, otherMatchKeys(merged, key));
    if (duplicate) {
      return reject(duplicate);
    }
    return apply(
      copies.map((row) => ({ type: "update", row, patch: { ...policy, enabled: row.enabled } })),
    );
  };
}

export function removePolicy(key: string): PolicyEdit {
  return (lists) => {
    const copies = copiesOf(merge(lists), key);
    if (copies.length === 0) {
      return reject(POLICY_ERRORS.gone);
    }
    return apply(copies.map((row) => ({ type: "delete", row })));
  };
}

function orderBetween(rows: PolicyRow[], index: number): number {
  const before = rows[index - 1]?._order;
  const after = rows[index + 1]?._order;
  if (before === undefined) {
    return (after ?? 0) - 1;
  }
  return after === undefined ? before + 1 : (before + after) / 2;
}

export function reorderPolicies(fromKey: string, toKey: string): PolicyEdit {
  return (lists) => {
    const merged = merge(lists);
    const from = merged.findIndex((m) => m.key === fromKey);
    const to = merged.findIndex((m) => m.key === toKey);
    if (from < 0 || to < 0) {
      return reject(POLICY_ERRORS.gone);
    }
    const moved = merged[from];
    return apply(
      movePolicy(merged, lists, from, to).flatMap(({ env, rows }): PolicyOp[] => {
        const row = moved[env];
        const index = rows.findIndex((r) => r === row);
        return row ? [{ type: "update", row, patch: { _order: orderBetween(rows, index) } }] : [];
      }),
    );
  };
}
