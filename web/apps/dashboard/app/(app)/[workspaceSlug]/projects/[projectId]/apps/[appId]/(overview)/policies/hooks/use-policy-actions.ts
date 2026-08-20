"use client";

import { collection } from "@/lib/collections";
import {
  type PolicyRow,
  findPolicyByName,
  nextPolicyOrder,
  reorderPolicies,
  rowKey,
} from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
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
      const environmentId = envIdFor(env);
      const policy = policyInEnv(merged, key, env);
      if (!policy) {
        return;
      }
      collection.policies.update(rowKey(environmentId, policy.id), (draft) => {
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
      const name = (prodPolicy ?? previewPolicy)?.name;
      if (!name) {
        return;
      }
      const targets = [
        { envId: envAId, policy: prodPolicy, existing: editing?.envA },
        { envId: envBId, policy: previewPolicy, existing: editing?.envB },
      ].filter((t) => t.envId);

      const updates: { key: string; envId: string; policy: Policy | null }[] = [];
      const insertRows: PolicyRow[] = [];

      for (const target of targets) {
        const existingRow = editing ? target.existing : findPolicyByName(target.envId, name);
        if (existingRow) {
          updates.push({
            key: rowKey(target.envId, existingRow.id),
            envId: target.envId,
            policy: target.policy,
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
        collection.policies.update(
          updates.map((u) => u.key),
          (drafts) => {
            for (let i = 0; i < drafts.length; i++) {
              const { envId, policy } = updates[i];
              if (policy) {
                // The form id can belong to the copy in the other
                // environment. Each row keeps the server id it has.
                const { id: _formId, ...fields } = policy;
                Object.assign(drafts[i], fields, {
                  environmentId: envId,
                  projectId,
                  appId,
                  enabled: true,
                });
              } else {
                drafts[i].enabled = false;
              }
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
        .filter((k): k is string => k !== undefined);
      if (keys.length > 0) {
        collection.policies.delete(keys);
      }
    },
    [envIdFor, merged],
  );

  return { toggleEnv, addToEnv, reorder, save, delete: remove };
}
