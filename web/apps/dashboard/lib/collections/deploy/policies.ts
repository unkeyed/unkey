"use client";

import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { match } from "@unkey/match";
import { toast } from "@unkey/ui";
import { queryClient, trpcClient } from "../client";
import { trackSave } from "./environment-settings";
import { type Policy, policySchema } from "./policies.schema";
import { parseEnvironmentIdFromWhere, validateEnvironmentIdInQuery } from "./utils";

/**
 * Whole-list reorder. Accepts a batch of (environmentId, policyIds) so a
 * single drag-drop that affects both envs (production + preview) emits ONE
 * toast and one trackSave, not one per env. Each entry is sent as its own
 * tRPC call but they're awaited together.
 */
export async function reorderPolicies(
  reorders: { environmentId: string; policyIds: string[] }[],
): Promise<void> {
  if (reorders.length === 0) {
    return;
  }
  const promise = Promise.all(
    reorders.map((r) =>
      trpcClient.deploy.environmentSettings.policies.reorder.mutate({
        environmentId: r.environmentId,
        policyIds: r.policyIds,
      }),
    ),
  );
  toast.promise(promise, {
    loading: "Reordering policies...",
    success: "Policies reordered",
    error: (err) => ({
      message: "Failed to reorder policies",
      description: err instanceof Error ? err.message : "Unknown error",
    }),
  });
  await trackSave(promise);
  for (const r of reorders) {
    queryClient.invalidateQueries({ queryKey: ["policies", r.environmentId] });
  }
}
/**
 * A row in the policies collection: a Policy plus the
 * environmentId it belongs to. Same policy id may exist in two envs
 * (production + preview), so the row key combines both.
 */
export type PolicyRow = Policy & {
  environmentId: string;
  // Preserves DB blob order. The collection stores rows in a Map keyed by
  // `${env}::${uuid}`, so iteration order is lexicographic by key — not blob
  // order. Stamp the blob index here and orderBy it in live queries so the
  // list reflects the actual order from the server.
  _order?: number;
};

export const rowKey = (environmentId: string, policyId: string) => `${environmentId}::${policyId}`;

/**
 * Gateway policies collection — one row per (environment, policy).
 *
 * IMPORTANT: All queries MUST filter by environmentId:
 * .where(({ p }) => eq(p.environmentId, environmentId))
 *
 * Mutations route by `policy.type` to the matching tRPC endpoint
 * (policies.keyauth.{create,update,delete} and policies.firewall.{create,
 * update,delete} today). To add a new policy type, extend
 * `policySchema` and add a branch in each dispatch* helper below.
 */
export const policies = createCollection<PolicyRow, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const environmentId = parseEnvironmentIdFromWhere(opts.where);
      return environmentId ? ["policies", environmentId] : ["policies"];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;
      validateEnvironmentIdInQuery(options?.where);
      const environmentId = parseEnvironmentIdFromWhere(options?.where);
      if (!environmentId) {
        throw new Error(
          "Query must include eq(collection.environmentId, environmentId) constraint",
        );
      }

      const result = await trpcClient.deploy.environmentSettings.policies.list.query({
        environmentId,
      });

      const rows: PolicyRow[] = result.policies.map((p, index) => ({
        ...p,
        environmentId,
        _order: index,
      }));
      return rows;
    },
    getKey: (row) => rowKey(row.environmentId, row.id),
    id: "policies",

    onInsert: async ({ transaction }) => {
      const mutations = transaction.mutations.map(async (m) => {
        const row = m.modified;
        // Re-validate before sending — collection.insert() accepts the row type,
        // but we want a hard guarantee the wire payload matches the canonical schema.
        const policy = policySchema.parse(stripEnv(row));
        return dispatchCreate(row.environmentId, policy);
      });
      const all = Promise.all(mutations);
      const plural = mutations.length > 1;
      toast.promise(all, {
        loading: plural ? "Adding policies..." : "Adding policy...",
        success: plural ? "Policies added" : "Policy added",
        error: (err) => ({
          message: plural ? "Failed to add policies" : "Failed to add policy",
          description: err instanceof Error ? err.message : "Unknown error",
        }),
      });
      await trackSave(all);
    },

    onUpdate: async ({ transaction }) => {
      const mutations = transaction.mutations.map(async (m) => {
        const row = m.modified;
        const policy = policySchema.parse(stripEnv(row));
        return dispatchUpdate(row.environmentId, policy);
      });
      const all = Promise.all(mutations);
      const plural = mutations.length > 1;
      toast.promise(all, {
        loading: plural ? "Updating policies..." : "Updating policy...",
        success: plural ? "Policies updated" : "Policy updated",
        error: (err) => ({
          message: plural ? "Failed to update policies" : "Failed to update policy",
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
      const plural = mutations.length > 1;
      toast.promise(all, {
        loading: plural ? "Deleting policies..." : "Deleting policy...",
        success: plural ? "Policies deleted" : "Policy deleted",
        error: (err) => ({
          message: plural ? "Failed to delete policies" : "Failed to delete policy",
          description: err instanceof Error ? err.message : "Unknown error",
        }),
      });
      await trackSave(all);
    },
  }),
);

/**
 * Returns the next `_order` value for a new policy in the given environment.
 * Scans the current collection state so optimistic inserts land at the end
 * without a flash of wrong ordering.
 */
export function nextPolicyOrder(environmentId: string): number {
  let max = -1;
  for (const [key, row] of policies.state) {
    if (key.startsWith(`${environmentId}::`)) {
      max = Math.max(max, row._order ?? -1);
    }
  }
  return max + 1;
}

// Returns `unknown` on purpose: every caller re-runs `policySchema.parse`
// which is the one source of truth for the Policy shape. This avoids
// fighting TS over discriminated-union narrowing across destructure-and-spread.
function stripEnv(row: PolicyRow): unknown {
  const { environmentId: _envId, _order: _o, ...policy } = row;
  return policy;
}

// ── Per-type dispatch ───────────────────────────────────────────────────
//
// Each branch maps a policy variant to its dedicated tRPC endpoint. `match`
// is exhaustive on `policy.type` — TS will complain when a new variant is
// added to `policySchema` without wiring it here.

function dispatchCreate(environmentId: string, policy: Policy): Promise<unknown> {
  return match(policy)
    .with({ type: "keyauth" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.keyauth.create.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "ratelimit" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.ratelimit.create.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "firewall" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.firewall.create.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "openapi" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.openapi.create.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "logging" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.logging.create.mutate({
        environmentId,
        policy: p,
      }),
    )
    .exhaustive();
}

function dispatchUpdate(environmentId: string, policy: Policy): Promise<unknown> {
  return match(policy)
    .with({ type: "keyauth" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.keyauth.update.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "ratelimit" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.ratelimit.update.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "firewall" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.firewall.update.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "openapi" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.openapi.update.mutate({
        environmentId,
        policy: p,
      }),
    )
    .with({ type: "logging" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.logging.update.mutate({
        environmentId,
        policy: p,
      }),
    )
    .exhaustive();
}

function dispatchDelete(environmentId: string, policy: Policy): Promise<unknown> {
  return match(policy)
    .with({ type: "keyauth" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.keyauth.delete.mutate({
        environmentId,
        policyId: p.id,
      }),
    )
    .with({ type: "ratelimit" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.ratelimit.delete.mutate({
        environmentId,
        policyId: p.id,
      }),
    )
    .with({ type: "firewall" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.firewall.delete.mutate({
        environmentId,
        policyId: p.id,
      }),
    )
    .with({ type: "openapi" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.openapi.delete.mutate({
        environmentId,
        policyId: p.id,
      }),
    )
    .with({ type: "logging" }, (p) =>
      trpcClient.deploy.environmentSettings.policies.logging.delete.mutate({
        environmentId,
        policyId: p.id,
      }),
    )
    .exhaustive();
}
