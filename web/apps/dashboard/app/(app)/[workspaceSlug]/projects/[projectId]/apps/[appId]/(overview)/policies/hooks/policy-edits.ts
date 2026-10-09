import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import type { PolicyEdit, PolicyEditResult, PolicyLists } from "@/lib/collections/deploy/policies";
import { POLICY_LIMITS, type PolicyInput } from "@/lib/collections/deploy/policies.schema";
import {
  type Env,
  type MergedPolicy,
  type PolicyRowKey,
  mergePolicies,
  policyInEnv,
} from "../components/list/merge";
import { movePolicy } from "../components/list/reorder";

const AT_CAPACITY = `An environment holds at most ${POLICY_LIMITS.maxPolicies} policies.`;
const GONE = "This policy no longer exists.";
const SHARED_NAME = "Another policy of this type has the same name. Rename one of them first.";

type Lists = Partial<Record<Env, PolicyInput[]>>;

const reject = (message: string): PolicyEditResult => ({ type: "reject", message });

function write(envs: readonly Env[], rows: (env: Env) => PolicyInput[]): PolicyEditResult {
  const lists: Lists = {};
  for (const env of envs) {
    lists[env] = rows(env);
  }
  return { type: "write", lists };
}

const merge = (lists: PolicyLists): MergedPolicy[] =>
  mergePolicies(lists.production, lists.preview);

const envsWith = (merged: MergedPolicy[], key: PolicyRowKey): Env[] =>
  ENVIRONMENT_KINDS.filter((env) => policyInEnv(merged, key, env) !== null);

export function insertPolicy(policy: PolicyInput, targets: readonly Env[]): PolicyEdit {
  return (lists) => {
    if (targets.some((env) => lists[env].length >= POLICY_LIMITS.maxPolicies)) {
      return reject(AT_CAPACITY);
    }
    return write(targets, (env) => [...lists[env], policy]);
  };
}

export function setEnabled(key: PolicyRowKey, env: Env, enabled: boolean): PolicyEdit {
  return (lists) => {
    const merged = merge(lists);
    const policy = policyInEnv(merged, key, env);
    if (policy) {
      return write([env], () =>
        lists[env].map((row) => (row.id === policy.id ? { ...row, enabled } : row)),
      );
    }
    const source = policyInEnv(merged, key, env === "production" ? "preview" : "production");
    if (!source) {
      return reject(GONE);
    }
    if (!enabled) {
      return write([], () => []);
    }
    if (lists[env].length >= POLICY_LIMITS.maxPolicies) {
      return reject(AT_CAPACITY);
    }
    const copy = { ...source, enabled };
    const next = { ...lists, [env]: [...lists[env], copy] };
    if (policyInEnv(merge(next), key, env) !== copy) {
      return reject(SHARED_NAME);
    }
    return write([env], () => next[env]);
  };
}

export function updatePolicy(key: PolicyRowKey, policy: PolicyInput): PolicyEdit {
  return (lists) => {
    const merged = merge(lists);
    const envs = envsWith(merged, key);
    if (envs.length === 0) {
      return reject(GONE);
    }
    return write(envs, (env) => {
      const id = policyInEnv(merged, key, env)?.id;
      return lists[env].map((row) => (row.id === id ? { ...policy, enabled: row.enabled } : row));
    });
  };
}

export function removePolicy(key: PolicyRowKey): PolicyEdit {
  return (lists) => {
    const merged = merge(lists);
    const envs = envsWith(merged, key);
    if (envs.length === 0) {
      return reject(GONE);
    }
    return write(envs, (env) => {
      const id = policyInEnv(merged, key, env)?.id;
      return lists[env].filter((row) => row.id !== id);
    });
  };
}

export function reorderPolicies(fromKey: PolicyRowKey, toKey: PolicyRowKey): PolicyEdit {
  return (lists) => {
    const merged = merge(lists);
    const from = merged.findIndex((m) => m.key === fromKey);
    const to = merged.findIndex((m) => m.key === toKey);
    if (from < 0 || to < 0) {
      return reject(GONE);
    }
    const moved = movePolicy(merged, lists, from, to);
    return write(
      moved.map(({ env }) => env),
      (env) => moved.find((m) => m.env === env)?.rows ?? lists[env],
    );
  };
}
