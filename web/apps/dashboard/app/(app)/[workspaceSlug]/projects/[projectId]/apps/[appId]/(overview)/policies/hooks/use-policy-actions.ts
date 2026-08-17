"use client";

import { collection } from "@/lib/collections";
import {
  type PolicyRow,
  nextPolicyOrder,
  reorderPolicies,
  rowKey,
} from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { useCallback } from "react";

type Args = { envAId: string; envBId: string };
type Env = "envA" | "envB";

export type PolicyActions = {
  toggleEnv: (id: string, env: Env) => void;
  addToEnv: (id: string, env: Env) => void;
  reorder: (envs: Env[], orderedIds: string[]) => void;
  save: (prodPolicy: Policy | null, previewPolicy: Policy | null) => void;
  delete: (id: string) => void;
};

/**
 * Per-row mutation handlers under the LWW model. All callbacks write directly
 * to the policies collection (or call `reorderPolicies`).
 */
export function usePolicyActions({ envAId, envBId }: Args): PolicyActions {
  const envIdFor = useCallback((env: Env) => (env === "envA" ? envAId : envBId), [envAId, envBId]);

  const toggleEnv = useCallback(
    (id: string, env: Env) => {
      const key = rowKey(envIdFor(env), id);
      if (!collection.policies.get(key)) {
        return;
      }
      collection.policies.update(key, (draft) => {
        draft.enabled = !draft.enabled;
      });
    },
    [envIdFor],
  );

  const addToEnv = useCallback(
    (id: string, env: Env) => {
      const targetEnvId = envIdFor(env);
      const sourceEnvId = env === "envA" ? envBId : envAId;
      if (!targetEnvId || !sourceEnvId) {
        return;
      }
      const sourceRow = collection.policies.get(rowKey(sourceEnvId, id));
      if (!sourceRow) {
        return;
      }
      const { environmentId: _e, _order: _o, ...sourcePolicy } = sourceRow;
      collection.policies.insert({
        ...(sourcePolicy as Policy),
        environmentId: targetEnvId,
        enabled: false,
        _order: nextPolicyOrder(targetEnvId),
      });
    },
    [envAId, envBId, envIdFor],
  );

  const reorder = useCallback(
    (envs: Env[], orderedIds: string[]) => {
      const reorders = envs
        .map((env) => ({ environmentId: envIdFor(env), policyIds: orderedIds }))
        .filter((r) => r.environmentId !== "");
      reorderPolicies(reorders);
    },
    [envIdFor],
  );

  /** Batched upsert across both envs. Existing rows are updated, missing ones inserted. */
  const save = useCallback(
    (prodPolicy: Policy | null, previewPolicy: Policy | null) => {
      const id = (prodPolicy ?? previewPolicy)?.id;
      if (!id) {
        return;
      }
      const targets = [
        { envId: envAId, policy: prodPolicy },
        { envId: envBId, policy: previewPolicy },
      ].filter((t) => t.envId);

      const updateKeys: string[] = [];
      const updateTargets: typeof targets = [];
      const insertRows: PolicyRow[] = [];

      for (const target of targets) {
        const key = rowKey(target.envId, id);
        if (collection.policies.get(key)) {
          updateKeys.push(key);
          updateTargets.push(target);
        } else if (target.policy) {
          insertRows.push({
            ...target.policy,
            environmentId: target.envId,
            _order: nextPolicyOrder(target.envId),
          });
        }
      }

      if (updateKeys.length > 0) {
        collection.policies.update(updateKeys, (drafts) => {
          for (let i = 0; i < drafts.length; i++) {
            const { envId, policy } = updateTargets[i];
            if (policy) {
              Object.assign(drafts[i], policy, { environmentId: envId, enabled: true });
            } else {
              drafts[i].enabled = false;
            }
          }
        });
      }

      if (insertRows.length > 0) {
        collection.policies.insert(insertRows);
      }
    },
    [envAId, envBId],
  );

  const remove = useCallback(
    (id: string) => {
      const keys = [envAId, envBId]
        .filter((envId) => envId && collection.policies.get(rowKey(envId, id)))
        .map((envId) => rowKey(envId, id));
      if (keys.length > 0) {
        collection.policies.delete(keys);
      }
    },
    [envAId, envBId],
  );

  return { toggleEnv, addToEnv, reorder, save, delete: remove };
}
