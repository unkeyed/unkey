"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import { type PolicyRow, replacePolicyLists, rowKey } from "@/lib/collections/deploy/policies";
import {
  POLICY_LIMITS,
  type Policy,
  policyMatchKey,
} from "@/lib/collections/deploy/policies.schema";
import { toast } from "@unkey/ui";
import { useCallback } from "react";
import { type Env, type MergedPolicy, policyInEnv } from "../components/list/merge";

const AT_CAPACITY = `An environment holds at most ${POLICY_LIMITS.maxPolicies} policies.`;

type Args = {
  productionId: string;
  previewId: string;
  projectId: string;
  appId: string;
  merged: MergedPolicy[];
  rowsByEnv: Record<Env, PolicyRow[]>;
};

export type PolicyActions = {
  toggleEnv: (key: string, env: Env) => void;
  addToEnv: (key: string, env: Env) => void;
  reorder: (envs: Env[], rowsByEnv: Partial<Record<Env, PolicyRow[]>>) => void;
  save: (prodPolicy: Policy | null, previewPolicy: Policy | null, editing?: MergedPolicy) => void;
  delete: (key: string) => void;
};

/**
 * An edit goes through the collection, which has `gateway.updatePolicy` behind
 * it. Insert, delete and reorder have no endpoint of their own, so they replace
 * an environment's whole list.
 *
 * That replace is last write wins, and neither endpoint takes a version: a
 * policy another tab added since this page loaded is dropped by it.
 */
export function usePolicyActions({
  productionId,
  previewId,
  projectId,
  appId,
  merged,
  rowsByEnv,
}: Args): PolicyActions {
  const envIdFor = useCallback(
    (env: Env) => (env === "production" ? productionId : previewId),
    [productionId, previewId],
  );

  const toggleEnv = useCallback(
    (key: string, env: Env) => {
      const policy = policyInEnv(merged, key, env);
      if (!policy) {
        return;
      }
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
      const source = policyInEnv(merged, key, env === "production" ? "preview" : "production");
      if (!source) {
        return;
      }
      const current = rowsByEnv[env];
      if (current.length >= POLICY_LIMITS.maxPolicies) {
        toast.error(AT_CAPACITY);
        return;
      }
      replacePolicyLists(
        [
          {
            environmentId: targetEnvId,
            projectId,
            appId,
            policies: [
              ...current,
              { ...source, environmentId: targetEnvId, projectId, appId, enabled: false },
            ],
          },
        ],
        {
          loading: "Adding policy...",
          success: "Policy added",
          error: "Failed to add policy",
        },
      );
    },
    [envIdFor, projectId, appId, merged, rowsByEnv],
  );

  const reorder = useCallback(
    (envs: Env[], reordered: Partial<Record<Env, PolicyRow[]>>) => {
      replacePolicyLists(
        envs
          .map((env) => ({
            environmentId: envIdFor(env),
            projectId,
            appId,
            policies: reordered[env] ?? [],
          }))
          .filter((r) => r.environmentId !== "" && r.policies.length > 0),
        {
          loading: "Reordering policies...",
          success: "Policies reordered",
          error: "Failed to reorder policies",
        },
      );
    },
    [envIdFor, projectId, appId],
  );

  /**
   * `editing` carries the row the panel opened, so an edit resolves its target
   * by id. Looking it up by the submitted name would miss on a rename and
   * append a second copy.
   */
  const save = useCallback(
    (prodPolicy: Policy | null, previewPolicy: Policy | null, editing?: MergedPolicy) => {
      const submitted = prodPolicy ?? previewPolicy;
      if (!submitted) {
        return;
      }
      const submittedMatchKey = policyMatchKey(submitted.type, submitted.name);
      const targets = [
        {
          env: "production" as const,
          envId: productionId,
          policy: prodPolicy,
          existing: editing?.production,
        },
        {
          env: "preview" as const,
          envId: previewId,
          policy: previewPolicy,
          existing: editing?.preview,
        },
      ].filter((t) => t.envId);

      const updates: { key: string; enabled: boolean }[] = [];
      const appends: Parameters<typeof replacePolicyLists>[0] = [];

      for (const target of targets) {
        const existingRow = editing
          ? target.existing
          : rowsByEnv[target.env].find((r) => policyMatchKey(r.type, r.name) === submittedMatchKey);
        if (existingRow) {
          updates.push({
            key: rowKey(target.envId, existingRow.id),
            enabled: target.policy !== null,
          });
        } else if (target.policy) {
          appends.push({
            environmentId: target.envId,
            projectId,
            appId,
            policies: [
              ...rowsByEnv[target.env],
              { ...target.policy, environmentId: target.envId, projectId, appId },
            ],
          });
        }
      }

      if (appends.some((a) => a.policies.length > POLICY_LIMITS.maxPolicies)) {
        toast.error(AT_CAPACITY);
        return;
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
              Object.assign(drafts[i], fields, { enabled: updates[i].enabled });
            }
          },
        );
      }

      replacePolicyLists(appends, {
        loading: "Adding policy...",
        success: "Policy added",
        error: "Failed to add policy",
      });
    },
    [productionId, previewId, projectId, appId, rowsByEnv],
  );

  const remove = useCallback(
    (key: string) => {
      replacePolicyLists(
        ENVIRONMENT_KINDS.flatMap((env) => {
          const policy = policyInEnv(merged, key, env);
          const envId = envIdFor(env);
          if (!policy || !envId) {
            return [];
          }
          return [
            {
              environmentId: envId,
              projectId,
              appId,
              policies: rowsByEnv[env].filter((r) => r.id !== policy.id),
            },
          ];
        }),
        {
          loading: "Deleting policy...",
          success: "Policy deleted",
          error: "Failed to delete policy",
        },
      );
    },
    [envIdFor, projectId, appId, merged, rowsByEnv],
  );

  return { toggleEnv, addToEnv, reorder, save, delete: remove };
}
