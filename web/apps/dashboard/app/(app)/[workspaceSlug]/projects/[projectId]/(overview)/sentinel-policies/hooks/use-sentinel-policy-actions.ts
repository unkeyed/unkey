"use client";

import { collection } from "@/lib/collections";
import {
  type SentinelPolicyRow,
  nextSentinelPolicyOrder,
  reorderSentinelPolicies,
  rowKey,
} from "@/lib/collections/deploy/sentinel-policies";
import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { useCallback } from "react";
import {
  type PolicyFormValues,
  resolveTargetEnvs,
  toSentinelPolicy,
} from "../components/add-panel/schema";

type Args = { envAId: string; envBId: string; envASlug: string; envBSlug: string };
type Env = "envA" | "envB";

export type SentinelPolicyActions = {
  toggleEnv: (id: string, env: Env) => void;
  addToEnv: (id: string, env: Env) => void;
  reorder: (envs: Env[], orderedIds: string[]) => void;
  save: (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => void;
  saveFromForm: (values: PolicyFormValues | PolicyFormValues[]) => void;
  delete: (id: string) => void;
};

/**
 * Per-row mutation handlers under the LWW model. All callbacks write directly
 * to the sentinelPolicies collection (or call `reorderSentinelPolicies`).
 */
export function useSentinelPolicyActions({
  envAId,
  envBId,
  envASlug,
  envBSlug,
}: Args): SentinelPolicyActions {
  const envIdFor = useCallback((env: Env) => (env === "envA" ? envAId : envBId), [envAId, envBId]);

  const toggleEnv = useCallback(
    (id: string, env: Env) => {
      const key = rowKey(envIdFor(env), id);
      if (!collection.sentinelPolicies.get(key)) {
        return;
      }
      collection.sentinelPolicies.update(key, (draft) => {
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
      const sourceRow = collection.sentinelPolicies.get(rowKey(sourceEnvId, id));
      if (!sourceRow) {
        return;
      }
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
    (envs: Env[], orderedIds: string[]) => {
      const reorders = envs
        .map((env) => ({ environmentId: envIdFor(env), policyIds: orderedIds }))
        .filter((r) => r.environmentId !== "");
      reorderSentinelPolicies(reorders);
    },
    [envIdFor],
  );

  /** Batched upsert across both envs. Existing rows are updated, missing ones inserted. */
  const save = useCallback(
    (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => {
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
      const insertRows: SentinelPolicyRow[] = [];

      for (const target of targets) {
        const key = rowKey(target.envId, id);
        if (collection.sentinelPolicies.get(key)) {
          updateKeys.push(key);
          updateTargets.push(target);
        } else if (target.policy) {
          insertRows.push({
            ...target.policy,
            environmentId: target.envId,
            _order: nextSentinelPolicyOrder(target.envId),
          });
        }
      }

      if (updateKeys.length > 0) {
        collection.sentinelPolicies.update(updateKeys, (drafts) => {
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
        collection.sentinelPolicies.insert(insertRows);
      }
    },
    [envAId, envBId],
  );

  const remove = useCallback(
    (id: string) => {
      const keys = [envAId, envBId]
        .filter((envId) => envId && collection.sentinelPolicies.get(rowKey(envId, id)))
        .map((envId) => rowKey(envId, id));
      if (keys.length > 0) {
        collection.sentinelPolicies.delete(keys);
      }
    },
    [envAId, envBId],
  );

  const saveFromForm = useCallback(
    (values: PolicyFormValues | PolicyFormValues[]) => {
      const items = Array.isArray(values) ? values : [values];
      const insertRows: SentinelPolicyRow[] = [];
      // `nextSentinelPolicyOrder` reads the collection, which hasn't been
      // updated yet within this loop, so every call returns the same slot.
      // Track a per-env offset so batched inserts land in distinct positions.
      const orderOffsets = new Map<string, number>();

      for (const v of items) {
        const policy = toSentinelPolicy(v);
        const { envA, envB } = resolveTargetEnvs(v.environmentId, envASlug, envBSlug);
        const envIds = [envA ? envAId : "", envB ? envBId : ""].filter(Boolean);
        for (const envId of envIds) {
          const offset = orderOffsets.get(envId) ?? 0;
          insertRows.push({
            ...policy,
            enabled: true,
            environmentId: envId,
            _order: nextSentinelPolicyOrder(envId) + offset,
          });
          orderOffsets.set(envId, offset + 1);
        }
      }

      if (insertRows.length > 0) {
        collection.sentinelPolicies.insert(insertRows);
      }
    },
    [envAId, envBId, envASlug, envBSlug],
  );

  return { toggleEnv, addToEnv, reorder, save, saveFromForm, delete: remove };
}
