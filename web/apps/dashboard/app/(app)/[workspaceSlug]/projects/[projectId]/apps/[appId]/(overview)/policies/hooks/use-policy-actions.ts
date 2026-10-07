"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import {
  type PolicyListReplacement,
  type PolicyRow,
  replacePolicyLists,
  rowKey,
} from "@/lib/collections/deploy/policies";
import { POLICY_LIMITS, type Policy } from "@/lib/collections/deploy/policies.schema";
import { toast } from "@unkey/ui";
import { useCallback } from "react";
import { type Env, type MergedPolicy, policyInEnv } from "../components/list/merge";
import { movePolicy } from "../components/list/reorder";
import type { PoliciesData } from "./use-policies-data";

const AT_CAPACITY = `An environment holds at most ${POLICY_LIMITS.maxPolicies} policies.`;

type Args = Pick<PoliciesData, "envs" | "merged" | "rowsByEnv" | "canWrite"> & {
  projectId: string;
  appId: string;
};

export type PolicyActions = {
  toggleEnv: (key: string, env: Env) => void;
  addToEnv: (key: string, env: Env) => void;
  reorder: (from: number, to: number) => void;
  save: (policy: Policy, envs: readonly Env[]) => Promise<boolean>;
  update: (policy: Policy, editing: MergedPolicy) => void;
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
  envs,
  projectId,
  appId,
  merged,
  rowsByEnv,
  canWrite,
}: Args): PolicyActions {
  const listFor = useCallback(
    (env: Env, policies: (Policy | PolicyRow)[]): PolicyListReplacement | null => {
      const environmentId = envs[env].id;
      if (!environmentId) {
        return null;
      }
      return {
        environmentId,
        projectId,
        appId,
        policies: policies.map((p) => ({ ...p, environmentId, projectId, appId })),
      };
    },
    [envs, projectId, appId],
  );

  const toggleEnv = useCallback(
    (key: string, env: Env) => {
      const policy = policyInEnv(merged, key, env);
      if (!policy) {
        return;
      }
      const rowId = rowKey(envs[env].id, policy.id);
      if (!collection.policies.get(rowId)) {
        return;
      }
      collection.policies.update(rowId, (draft) => {
        draft.enabled = !draft.enabled;
      });
    },
    [envs, merged],
  );

  const addToEnv = useCallback(
    (key: string, env: Env) => {
      if (!canWrite) {
        return;
      }
      const source = policyInEnv(merged, key, env === "production" ? "preview" : "production");
      if (!source) {
        return;
      }
      const current = rowsByEnv[env];
      const list = listFor(env, [...current, { ...source, enabled: false }]);
      if (!list) {
        return;
      }
      if (current.length >= POLICY_LIMITS.maxPolicies) {
        toast.error(AT_CAPACITY);
        return;
      }
      replacePolicyLists([list], {
        loading: "Adding policy...",
        success: `Added to ${envs[env].slug}`,
        error: "Failed to add policy",
      });
    },
    [canWrite, envs, merged, rowsByEnv, listFor],
  );

  const reorder = useCallback(
    (from: number, to: number) => {
      if (!canWrite) {
        return;
      }
      replacePolicyLists(
        movePolicy(merged, rowsByEnv, from, to)
          .map(({ env, rows }) => listFor(env, rows))
          .filter((list) => list !== null),
        {
          loading: "Reordering policies...",
          success: "Policies reordered",
          error: "Failed to reorder policies",
        },
      );
    },
    [canWrite, merged, rowsByEnv, listFor],
  );

  const save = useCallback(
    async (policy: Policy, targets: readonly Env[]) => {
      if (!canWrite || targets.length === 0) {
        return false;
      }
      const appends = targets
        .map((env) => listFor(env, [...rowsByEnv[env], policy]))
        .filter((list) => list !== null);

      if (appends.length === 0) {
        toast.error("Couldn't find the environment. Reload the page and try again.");
        return false;
      }

      if (appends.some((a) => a.policies.length > POLICY_LIMITS.maxPolicies)) {
        toast.error(AT_CAPACITY);
        return false;
      }

      try {
        await replacePolicyLists(appends, {
          loading: "Adding policy...",
          success: "Policy added",
          error: "Failed to add policy",
        });
        return true;
      } catch {
        return false;
      }
    },
    [canWrite, rowsByEnv, listFor],
  );

  /**
   * A merged row is one policy: the rule and the name reach every environment
   * it exists in. Each copy keeps its own server id and its own `enabled`.
   */
  const update = useCallback(
    (policy: Policy, editing: MergedPolicy) => {
      const keys = ENVIRONMENT_KINDS.flatMap((env) => {
        const row = editing[env];
        const envId = envs[env].id;
        return row && envId ? [rowKey(envId, row.id)] : [];
      });
      if (keys.length === 0) {
        return;
      }
      const { id: _formId, enabled: _enabled, ...fields } = policy;
      collection.policies.update(keys, (drafts) => {
        for (const draft of drafts) {
          Object.assign(draft, fields);
        }
      });
    },
    [envs],
  );

  const remove = useCallback(
    (key: string) => {
      if (!canWrite) {
        return;
      }
      replacePolicyLists(
        ENVIRONMENT_KINDS.flatMap((env) => {
          const policy = policyInEnv(merged, key, env);
          const list =
            policy &&
            listFor(
              env,
              rowsByEnv[env].filter((r) => r.id !== policy.id),
            );
          return list ? [list] : [];
        }),
        {
          loading: "Deleting policy...",
          success: "Policy deleted",
          error: "Failed to delete policy",
        },
      );
    },
    [canWrite, merged, rowsByEnv, listFor],
  );

  return { toggleEnv, addToEnv, reorder, save, update, delete: remove };
}
