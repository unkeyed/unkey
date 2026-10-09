"use client";

import {
  type PolicyEdit,
  type PolicyListToast,
  type PolicyScope,
  writePolicies,
} from "@/lib/collections/deploy/policies";
import type { PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { useMemo } from "react";
import type { Env, PolicyEnvs, PolicyRowKey } from "../components/list/merge";
import {
  insertPolicy,
  removePolicy,
  reorderPolicies,
  setEnabled,
  updatePolicy,
} from "./policy-edits";

type Args = { envs: PolicyEnvs; projectId: string; appId: string };

export type PolicyActions = {
  setEnabled: (key: PolicyRowKey, env: Env, enabled: boolean) => Promise<boolean>;
  reorder: (fromKey: PolicyRowKey, toKey: PolicyRowKey) => Promise<boolean>;
  save: (policy: PolicyInput, envs: readonly Env[]) => Promise<boolean>;
  update: (key: PolicyRowKey, policy: PolicyInput) => Promise<boolean>;
  delete: (key: PolicyRowKey) => Promise<boolean>;
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

export function usePolicyActions({ envs, projectId, appId }: Args): PolicyActions {
  return useMemo(() => {
    const scope: PolicyScope = {
      projectId,
      appId,
      environments: { production: envs.production.id, preview: envs.preview.id },
    };
    const run = (edit: PolicyEdit, labels: PolicyListToast) => writePolicies(scope, edit, labels);
    return {
      setEnabled: (key, env, enabled) => run(setEnabled(key, env, enabled), TOASTS.update),
      reorder: (fromKey, toKey) => run(reorderPolicies(fromKey, toKey), TOASTS.reorder),
      save: (policy, targets) => run(insertPolicy(policy, targets), TOASTS.add),
      update: (key, policy) => run(updatePolicy(key, policy), TOASTS.update),
      delete: (key) => run(removePolicy(key), TOASTS.delete),
    };
  }, [envs, projectId, appId]);
}
