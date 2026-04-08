"use client";

import { collection } from "@/lib/collections";
import {
  nextSentinelPolicyOrder,
  reorderSentinelPolicies,
  type SentinelPolicyRow,
} from "@/lib/collections/deploy/sentinel-policies";
import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { useCallback } from "react";

type Args = { envAId: string; envBId: string };

export type SentinelPolicyActions = {
  toggleEnv: (id: string, env: "envA" | "envB") => void;
  addToEnv: (id: string, env: "envA" | "envB") => void;
  reorder: (envs: ("envA" | "envB")[], orderedIds: string[]) => void;
  add: (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => void;
  save: (updated: SentinelPolicy) => void;
  delete: (id: string) => void;
};

/**
 * Per-row mutation handlers under the LWW model. All callbacks write directly
 * to the sentinelPolicies collection (or call `reorderSentinelPolicies`). 
 */
export function useSentinelPolicyActions({ envAId, envBId }: Args): SentinelPolicyActions {
  const envIdFor = useCallback(
    (env: "envA" | "envB") => (env === "envA" ? envAId : envBId),
    [envAId, envBId],
  );

  const toggleEnv = useCallback(
    (id: string, env: "envA" | "envB") => {
      const envId = envIdFor(env);
      if (!envId) return;
      const key = `${envId}::${id}`;
      if (!collection.sentinelPolicies.get(key)) return;
      collection.sentinelPolicies.update(key, (draft) => {
        draft.enabled = !draft.enabled;
      });
    },
    [envIdFor],
  );

  const addToEnv = useCallback(
    (id: string, env: "envA" | "envB") => {
      const targetEnvId = envIdFor(env);
      const sourceEnvId = env === "envA" ? envBId : envAId;
      if (!targetEnvId || !sourceEnvId) return;
      const sourceRow = collection.sentinelPolicies.get(`${sourceEnvId}::${id}`);
      if (!sourceRow) return;
      const { environmentId: _e, _order: _o, ...sourcePolicy } = sourceRow;
      collection.sentinelPolicies.insert({
        ...(sourcePolicy as SentinelPolicy),
        environmentId: targetEnvId,
        enabled: false,
        _order: nextSentinelPolicyOrder(targetEnvId),
      });
    },
    [envAId, envBId, envIdFor],
  );

  const reorder = useCallback(
    (envs: ("envA" | "envB")[], orderedIds: string[]) => {
      const reorders = envs
        .map((env) => ({ environmentId: envIdFor(env), policyIds: orderedIds }))
        .filter((r) => r.environmentId !== "");
      reorderSentinelPolicies(reorders);
    },
    [envIdFor],
  );

  const add = useCallback(
    (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => {
      const rows: SentinelPolicyRow[] = [];
      if (prodPolicy !== null && envAId) {
        rows.push({ ...prodPolicy, environmentId: envAId, _order: nextSentinelPolicyOrder(envAId) });
      }
      if (previewPolicy !== null && envBId) {
        rows.push({
          ...previewPolicy,
          environmentId: envBId,
          _order: nextSentinelPolicyOrder(envBId),
        });
      }
      if (rows.length > 0) collection.sentinelPolicies.insert(rows);
    },
    [envAId, envBId],
  );

  const save = useCallback(
    (updated: SentinelPolicy) => {
      for (const envId of [envAId, envBId]) {
        if (!envId) continue;
        const key = `${envId}::${updated.id}`;
        if (!collection.sentinelPolicies.get(key)) continue;
        collection.sentinelPolicies.update(key, (draft) => {
          // Preserve this environment's own enabled state. The caller passes
          // initialPolicy.enabled from one specific env, so spreading it would
          // clobber the other env's toggle.
          const currentEnabled = draft.enabled;
          Object.assign(draft, updated, { environmentId: envId, enabled: currentEnabled });
        });
      }
    },
    [envAId, envBId],
  );

  const remove = useCallback(
    (id: string) => {
      // Batch both env-rows into one transaction so the collection's onDelete
      // toast fires once, not twice (count is plural-aware).
      const keys: string[] = [];
      if (envAId && collection.sentinelPolicies.get(`${envAId}::${id}`)) {
        keys.push(`${envAId}::${id}`);
      }
      if (envBId && collection.sentinelPolicies.get(`${envBId}::${id}`)) {
        keys.push(`${envBId}::${id}`);
      }
      if (keys.length > 0) collection.sentinelPolicies.delete(keys);
    },
    [envAId, envBId],
  );

  return { toggleEnv, addToEnv, reorder, add, save, delete: remove };
}
