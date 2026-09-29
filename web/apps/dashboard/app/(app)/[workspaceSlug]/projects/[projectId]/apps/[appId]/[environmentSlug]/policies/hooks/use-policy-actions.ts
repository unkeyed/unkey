"use client";

import { collection } from "@/lib/collections";
import { type PolicyRow, replacePolicyLists, rowKey } from "@/lib/collections/deploy/policies";
import {
  POLICY_LIMITS,
  type Policy,
  policyMatchKey,
} from "@/lib/collections/deploy/policies.schema";
import { toast } from "@unkey/ui";
import { useCallback } from "react";
import type { Env, MergedPolicy } from "../components/list/merge";

const AT_CAPACITY = `An environment holds at most ${POLICY_LIMITS.maxPolicies} policies.`;

type Args = {
  env: Env;
  productionId: string;
  previewId: string;
  projectId: string;
  appId: string;
  rowsByEnv: Record<Env, PolicyRow[]>;
};

export type PolicyActions = {
  toggle: (id: string) => void;
  reorder: (rows: PolicyRow[]) => void;
  save: (prodPolicy: Policy | null, previewPolicy: Policy | null, editing?: MergedPolicy) => void;
  delete: (id: string) => void;
};

/**
 * An edit goes through the collection, which has `gateway.updatePolicy` behind
 * it. Insert, delete and reorder have no endpoint of their own, so they replace
 * an environment's whole list.
 *
 * That replace is last write wins, and neither endpoint takes a version: a
 * policy another tab added since this page loaded is dropped by it.
 *
 * Toggle, reorder and delete act on `env` only. `save` can reach the other
 * environment too, for the "also apply to" choice in the panel.
 */
export function usePolicyActions({
  env,
  productionId,
  previewId,
  projectId,
  appId,
  rowsByEnv,
}: Args): PolicyActions {
  const envIdFor = useCallback(
    (target: Env) => (target === "production" ? productionId : previewId),
    [productionId, previewId],
  );
  const envId = envIdFor(env);

  const toggle = useCallback(
    (id: string) => {
      const key = rowKey(envId, id);
      if (!collection.policies.get(key)) {
        return;
      }
      collection.policies.update(key, (draft) => {
        draft.enabled = !draft.enabled;
      });
    },
    [envId],
  );

  const reorder = useCallback(
    (rows: PolicyRow[]) => {
      if (!envId || rows.length === 0) {
        return;
      }
      replacePolicyLists([{ environmentId: envId, projectId, appId, policies: rows }], {
        loading: "Reordering policies...",
        success: "Policies reordered",
        error: "Failed to reorder policies",
      });
    },
    [envId, projectId, appId],
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
    (id: string) => {
      if (!envId) {
        return;
      }
      replacePolicyLists(
        [
          {
            environmentId: envId,
            projectId,
            appId,
            policies: rowsByEnv[env].filter((r) => r.id !== id),
          },
        ],
        {
          loading: "Deleting policy...",
          success: "Policy deleted",
          error: "Failed to delete policy",
        },
      );
    },
    [env, envId, projectId, appId, rowsByEnv],
  );

  return { toggle, reorder, save, delete: remove };
}
