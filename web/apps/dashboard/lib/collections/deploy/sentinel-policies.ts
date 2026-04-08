"use client";

import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { queryClient, trpcClient } from "../client";
import { trackSave } from "./environment-settings";
import { type SentinelPolicy, sentinelPolicySchema } from "./sentinel-policies.schema";
import { parseEnvironmentIdFromWhere, validateEnvironmentIdInQuery } from "./utils";

/**
 * A row in the sentinelPolicies collection: a SentinelPolicy plus the
 * environmentId it belongs to. Same policy id may exist in two envs
 * (production + preview), so the row key combines both.
 */
export type SentinelPolicyRow = SentinelPolicy & {
  environmentId: string;
};

const rowKey = (environmentId: string, policyId: string) => `${environmentId}::${policyId}`;

/**
 * Sentinel policies collection — one row per (environment, policy).
 *
 * IMPORTANT: All queries MUST filter by environmentId:
 * .where(({ p }) => eq(p.environmentId, environmentId))
 *
 * Mutations route by `policy.type` to the matching tRPC endpoint
 * (sentinel.keyauth.{create,update,delete} for now). To add a new policy
 * type, extend `sentinelPolicySchema` and add a branch in each handler.
 */
export const sentinelPolicies = createCollection<SentinelPolicyRow, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const environmentId = parseEnvironmentIdFromWhere(opts.where);
      return environmentId ? ["sentinelPolicies", environmentId] : ["sentinelPolicies"];
    },
    retry: 3,
    syncMode: "on-demand",
    refetchInterval: 30_000,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;
      validateEnvironmentIdInQuery(options?.where);
      const environmentId = parseEnvironmentIdFromWhere(options?.where);
      if (!environmentId) {
        throw new Error(
          "Query must include eq(collection.environmentId, environmentId) constraint",
        );
      }

      const result = await trpcClient.deploy.environmentSettings.sentinel.list.query({
        environmentId,
      });

      return result.policies.map((p) => ({ ...p, environmentId }));
    },
    getKey: (row) => rowKey(row.environmentId, row.id),
    id: "sentinelPolicies",

    onInsert: async ({ transaction }) => {
      const mutations = transaction.mutations.map(async (m) => {
        const row = m.modified;
        // Re-validate before sending — collection.insert() accepts the row type,
        // but we want a hard guarantee the wire payload matches the canonical schema.
        const policy = sentinelPolicySchema.parse(stripEnv(row));
        return dispatchCreate(row.environmentId, policy);
      });
      const all = Promise.all(mutations);
      toast.promise(all, {
        loading: "Adding sentinel policy...",
        success: "Sentinel policy added",
        error: (err) => ({
          message: "Failed to add sentinel policy",
          description: err instanceof Error ? err.message : "Unknown error",
        }),
      });
      await trackSave(all);
    },

    onUpdate: async ({ transaction }) => {
      const mutations = transaction.mutations.map(async (m) => {
        const row = m.modified;
        const policy = sentinelPolicySchema.parse(stripEnv(row));
        return dispatchUpdate(row.environmentId, policy);
      });
      const all = Promise.all(mutations);
      toast.promise(all, {
        loading: "Updating sentinel policy...",
        success: "Sentinel policy updated",
        error: (err) => ({
          message: "Failed to update sentinel policy",
          description: err instanceof Error ? err.message : "Unknown error",
        }),
      });
      await trackSave(all);
    },

    onDelete: async ({ transaction }) => {
      const mutations = transaction.mutations.map((m) => {
        const row = m.original;
        return dispatchDelete(row.environmentId, row);
      });
      const all = Promise.all(mutations);
      const count = mutations.length;
      toast.promise(all, {
        loading: `Deleting ${count === 1 ? "sentinel policy" : `${count} sentinel policies`}...`,
        success: `${count === 1 ? "Sentinel policy" : `${count} sentinel policies`} deleted`,
        error: (err) => ({
          message: "Failed to delete sentinel policy",
          description: err instanceof Error ? err.message : "Unknown error",
        }),
      });
      await trackSave(all);
    },
  }),
);

function stripEnv(row: SentinelPolicyRow): SentinelPolicy {
  const { environmentId: _envId, ...policy } = row;
  return policy as SentinelPolicy;
}

// ── Per-type dispatch ───────────────────────────────────────────────────
//
// Each branch maps a policy variant to its dedicated tRPC endpoint. The
// switch is exhaustive on `policy.type` — TS will complain when a new
// variant is added to `sentinelPolicySchema` without wiring it here.

function dispatchCreate(environmentId: string, policy: SentinelPolicy): Promise<unknown> {
  switch (policy.type) {
    case "keyauth":
      return trpcClient.deploy.environmentSettings.sentinel.keyauth.create.mutate({
        environmentId,
        policy,
      });
    default: {
      const _exhaustive: never = policy.type;
      throw new Error(`Unsupported sentinel policy type: ${_exhaustive}`);
    }
  }
}

function dispatchUpdate(environmentId: string, policy: SentinelPolicy): Promise<unknown> {
  switch (policy.type) {
    case "keyauth":
      return trpcClient.deploy.environmentSettings.sentinel.keyauth.update.mutate({
        environmentId,
        policy,
      });
    default: {
      const _exhaustive: never = policy.type;
      throw new Error(`Unsupported sentinel policy type: ${_exhaustive}`);
    }
  }
}

function dispatchDelete(environmentId: string, policy: SentinelPolicy): Promise<unknown> {
  switch (policy.type) {
    case "keyauth":
      return trpcClient.deploy.environmentSettings.sentinel.keyauth.delete.mutate({
        environmentId,
        policyId: policy.id,
      });
    default: {
      const _exhaustive: never = policy.type;
      throw new Error(`Unsupported sentinel policy type: ${_exhaustive}`);
    }
  }
}
