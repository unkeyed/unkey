"use client";

import {
  type PolicyListToast,
  type PolicyRow,
  persistPolicyMutations,
  policies,
  policyList,
  rowKey,
} from "@/lib/collections/deploy/policies";
import { POLICY_ERRORS, type PolicyInput } from "@/lib/collections/deploy/policies.schema";
import {
  type QueueStrategy,
  type Transaction,
  queueStrategy,
  usePacedMutations,
} from "@tanstack/react-db";
import { match } from "@unkey/match";
import { toast } from "@unkey/ui";
import { useState } from "react";
import {
  type PolicyEdit,
  type PolicyOp,
  insertPolicy,
  removePolicy,
  reorderPolicies,
  replacesList,
  setEnabled,
  updatePolicy,
} from "../model/edits";
import type { Env, PolicyEnvs } from "../model/merge";

type Args = { envs: PolicyEnvs; projectId: string; appId: string };

export type PolicyActions = {
  replacingList: boolean;
  setEnabled: (key: string, env: Env, enabled: boolean) => Promise<boolean>;
  reorder: (fromKey: string, toKey: string) => Promise<boolean>;
  save: (policy: PolicyInput, envs: readonly Env[]) => Promise<boolean>;
  update: (key: string, policy: PolicyInput) => Promise<boolean>;
  delete: (key: string) => Promise<boolean>;
};

const TOASTS = {
  add: { loading: "Adding policy...", success: "Policy added", error: "Failed to add policy" },
  update: {
    loading: "Updating policy...",
    success: "Policy updated",
    error: "Failed to update policy",
  },
  delete: {
    loading: "Deleting policy...",
    success: "Policy deleted",
    error: "Failed to delete policy",
  },
  reorder: {
    loading: "Reordering policies...",
    success: "Policies reordered",
    error: "Failed to reorder policies",
  },
} satisfies Record<string, PolicyListToast>;

function applyOps(ops: PolicyOp[]) {
  for (const op of ops) {
    const key = rowKey(op.row.environmentId, op.row.id);
    match(op)
      .with({ type: "insert" }, ({ row }) => policies.insert(row))
      .with({ type: "update" }, ({ patch }) =>
        policies.update(key, (draft) => {
          Object.assign(draft, patch);
        }),
      )
      .with({ type: "delete" }, () => policies.delete(key))
      .exhaustive();
  }
}

function usePolicyWrite(strategy: QueueStrategy, labels: PolicyListToast) {
  return usePacedMutations<PolicyOp[], PolicyRow>({
    strategy,
    onMutate: applyOps,
    mutationFn: ({ transaction }) => persistPolicyMutations(transaction.mutations, labels),
  });
}

export function usePolicyActions({ envs, projectId, appId }: Args): PolicyActions {
  const [strategy] = useState(() => queueStrategy());
  const add = usePolicyWrite(strategy, TOASTS.add);
  const update = usePolicyWrite(strategy, TOASTS.update);
  const remove = usePolicyWrite(strategy, TOASTS.delete);
  const reorder = usePolicyWrite(strategy, TOASTS.reorder);
  const [replacing, setReplacing] = useState<ReadonlySet<Transaction<PolicyRow>>>(new Set());

  const trackReplace = (tx: Transaction<PolicyRow>) => {
    setReplacing((prev) => new Set(prev).add(tx));
    void tx.isPersisted.promise
      .catch(() => undefined)
      .finally(() =>
        setReplacing((prev) => new Set([...prev].filter((pending) => pending !== tx))),
      );
  };

  const run = async (
    write: ReturnType<typeof usePolicyWrite>,
    edit: PolicyEdit,
  ): Promise<boolean> => {
    const production = envs.production.id;
    const preview = envs.preview.id;
    if (production === null || preview === null) {
      toast.error(POLICY_ERRORS.environmentMissing);
      return false;
    }
    const result = edit(
      { production: policyList(production), preview: policyList(preview) },
      { projectId, appId, environments: { production, preview } },
    );
    if (result.type === "reject") {
      toast.error(result.message);
      return false;
    }
    if (result.ops.length === 0) {
      return true;
    }
    const replaces = replacesList(result.ops);
    // An in-place write queued behind a list replace targets ids the replace retires.
    if (!replaces && replacing.size > 0) {
      return false;
    }
    const tx = write(result.ops);
    if (replaces) {
      trackReplace(tx);
    }
    return tx.isPersisted.promise.then(
      () => true,
      () => false,
    );
  };

  return {
    replacingList: replacing.size > 0,
    setEnabled: (key, env, enabled) => run(update, setEnabled(key, env, enabled)),
    reorder: (fromKey, toKey) => run(reorder, reorderPolicies(fromKey, toKey)),
    save: (policy, targets) => run(add, insertPolicy(policy, targets)),
    update: (key, policy) => run(update, updatePolicy(key, policy)),
    delete: (key) => run(remove, removePolicy(key)),
  };
}
