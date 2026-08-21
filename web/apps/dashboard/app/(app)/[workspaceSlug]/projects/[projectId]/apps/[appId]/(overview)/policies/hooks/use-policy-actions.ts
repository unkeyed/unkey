"use client";

import { collection } from "@/lib/collections";
import {
  type PolicyRow,
  clearOtherVariants,
  findPolicyByIdentity,
  nextPolicyOrder,
  policyCount,
  reorderPolicies,
  rowKey,
} from "@/lib/collections/deploy/policies";
import { POLICY_LIMITS, type Policy } from "@/lib/collections/deploy/policies.schema";
import { toast } from "@unkey/ui";
import { useCallback } from "react";
import { type MergedPolicy, policyInEnv } from "../components/list/merge";

type Args = {
  envAId: string;
  envBId: string;
  projectId: string;
  appId: string;
  merged: MergedPolicy[];
};
type Env = "envA" | "envB";

export type PolicyActions = {
  toggleEnv: (key: string, env: Env) => void;
  addToEnv: (key: string, env: Env) => void;
  reorder: (envs: Env[], rowsByEnv: Record<string, PolicyRow[]>) => void;
  save: (prodPolicy: Policy | null, previewPolicy: Policy | null, editing?: MergedPolicy) => void;
  delete: (key: string) => void;
};

/**
 * Per-row mutation handlers under the LWW model. `key` throughout is a merge
 * key from `merge.ts`, not a server policy id.
 */
export function usePolicyActions({
  envAId,
  envBId,
  projectId,
  appId,
  merged,
}: Args): PolicyActions {
  const envIdFor = useCallback((env: Env) => (env === "envA" ? envAId : envBId), [envAId, envBId]);

  const toggleEnv = useCallback(
    (key: string, env: Env) => {
      const policy = policyInEnv(merged, key, env);
      if (!policy) {
        return;
      }
      // `merged` is a render snapshot, and updating a dropped key throws.
      const rowId = rowKey(envIdFor(env), policy.id);
      if (!collection.policies.get(rowId)) {
        return;
      }
      collection.policies.update(rowId, (draft) => {
        draft.enabled = !draft.enabled;
      });
    },
    [envIdFor, merged],
  );

  const addToEnv = useCallback(
    (key: string, env: Env) => {
      const targetEnvId = envIdFor(env);
      if (!targetEnvId) {
        return;
      }
      const source = policyInEnv(merged, key, env === "envA" ? "envB" : "envA");
      if (!source) {
        return;
      }
      if (policyCount(targetEnvId) >= POLICY_LIMITS.maxPolicies) {
        toast.error(`An environment holds at most ${POLICY_LIMITS.maxPolicies} policies.`);
        return;
      }
      collection.policies.insert({
        ...source,
        environmentId: targetEnvId,
        projectId,
        appId,
        enabled: false,
        _order: nextPolicyOrder(targetEnvId),
      });
    },
    [envIdFor, projectId, appId, merged],
  );

  const reorder = useCallback(
    (envs: Env[], rowsByEnv: Record<string, PolicyRow[]>) => {
      const reorders = envs
        .map((env) => ({
          environmentId: envIdFor(env),
          projectId,
          appId,
          policies: rowsByEnv[env] ?? [],
        }))
        .filter((r) => r.environmentId !== "" && r.policies.length > 0);
      reorderPolicies(reorders);
    },
    [envIdFor, projectId, appId],
  );

  /**
   * Batched upsert across both envs. Existing rows are updated, missing ones
   * inserted.
   *
   * `editing` carries the row the panel opened, so an edit resolves its target
   * by id. Looking it up by the submitted name would miss on a rename and
   * insert a second copy under the old id, which collides on the collection
   * key.
   */
  const save = useCallback(
    (prodPolicy: Policy | null, previewPolicy: Policy | null, editing?: MergedPolicy) => {
      const submitted = prodPolicy ?? previewPolicy;
      if (!submitted) {
        return;
      }
      const targets = [
        { envId: envAId, policy: prodPolicy, existing: editing?.envA },
        { envId: envBId, policy: previewPolicy, existing: editing?.envB },
      ].filter((t) => t.envId);

      const updates: { key: string; enabled: boolean }[] = [];
      const insertRows: PolicyRow[] = [];

      for (const target of targets) {
        const existingRow = editing
          ? target.existing
          : findPolicyByIdentity(target.envId, submitted.type, submitted.name);
        if (existingRow) {
          updates.push({
            key: rowKey(target.envId, existingRow.id),
            enabled: target.policy !== null,
          });
        } else if (target.policy) {
          insertRows.push({
            ...target.policy,
            environmentId: target.envId,
            projectId,
            appId,
            _order: nextPolicyOrder(target.envId),
          });
        }
      }

      if (updates.length > 0) {
        // The form id can belong to the copy in the other environment. Each row
        // keeps the server id it has.
        const { id: _formId, ...fields } = submitted;
        collection.policies.update(
          updates.map((u) => u.key),
          (drafts) => {
            for (let i = 0; i < drafts.length; i++) {
              // A merged row is one policy: the rule and the name reach every
              // environment it exists in, and only `enabled` follows the panel's
              // choice. Renaming the selected copy alone would unpair the row.
              clearOtherVariants(drafts[i], submitted.type);
              Object.assign(drafts[i], fields, { enabled: updates[i].enabled });
            }
          },
        );
      }

      if (insertRows.length > 0) {
        collection.policies.insert(insertRows);
      }
    },
    [envAId, envBId, projectId, appId],
  );

  const remove = useCallback(
    (key: string) => {
      const keys = (["envA", "envB"] as const)
        .map((env) => {
          const policy = policyInEnv(merged, key, env);
          const envId = envIdFor(env);
          return policy && envId ? rowKey(envId, policy.id) : undefined;
        })
        .filter((k): k is string => k !== undefined)
        .filter((k) => collection.policies.get(k) !== undefined);
      if (keys.length > 0) {
        collection.policies.delete(keys);
      }
    },
    [envIdFor, merged],
  );

  return { toggleEnv, addToEnv, reorder, save, delete: remove };
}
