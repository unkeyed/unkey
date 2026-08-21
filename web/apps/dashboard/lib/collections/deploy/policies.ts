"use client";

import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { queryClient } from "../client";
import { trackSave } from "./environment-settings";
import { type Policy, fromWirePolicy } from "./policies.schema";
import { extractStringFilter } from "./utils";

/** A Policy plus the identifiers every gateway call is scoped by. */
export type PolicyRow = Policy & {
  environmentId: string;
  projectId: string;
  appId: string;
  // Collection keys sort lexicographically, not by evaluation order, so
  // carry the server's list index and orderBy it in live queries.
  _order?: number;
};

export const rowKey = (environmentId: string, policyId: string) => `${environmentId}::${policyId}`;

/**
 * Gateway policies collection. It holds one row for each (environment, policy).
 *
 * IMPORTANT: All queries MUST filter by projectId, appId, and environmentId.
 * `listPolicies` reads one environment, so each environment is its own subset
 * and its own request. The two environments of a page load in parallel.
 *
 * Only edits run through it. Insert, delete and reorder go through
 * `replacePolicyLists` instead.
 */
export const policies = createCollection<PolicyRow, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const environmentId = extractStringFilter(filters, "environmentId");
      return environmentId ? ["policies", environmentId] : ["policies"];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const projectId = extractStringFilter(filters, "projectId");
      const appId = extractStringFilter(filters, "appId");
      const environmentId = extractStringFilter(filters, "environmentId");

      if (!projectId || !appId || !environmentId) {
        throw new Error(
          "Query must include eq(collection.projectId, ...), eq(collection.appId, ...) and eq(collection.environmentId, ...) constraints",
        );
      }

      const result = await getUnkeyClient().gateway.listPolicies({
        project: projectId,
        app: appId,
        environment: environmentId,
      });

      return result.data.map(
        (p, index): PolicyRow => ({
          ...fromWirePolicy(p),
          environmentId,
          projectId,
          appId,
          _order: index,
        }),
      );
    },
    getKey: (row) => rowKey(row.environmentId, row.id),
    id: "policies",
    onUpdate: async ({ transaction }) => {
      const mutations = transaction.mutations.map((m) =>
        getUnkeyClient().gateway.updatePolicy({
          ...m.changes,
          project: m.modified.projectId,
          app: m.modified.appId,
          environment: m.modified.environmentId,
          policyId: m.modified.id,
        }),
      );
      const all = Promise.all(mutations);
      const plural = mutations.length > 1;
      toast.promise(all, {
        loading: plural ? "Updating policies..." : "Updating policy...",
        success: plural ? "Policies updated" : "Policy updated",
        error: (err) =>
          getErrorToast(err, plural ? "Failed to update policies" : "Failed to update policy"),
      });
      await trackSave(all);
    },
  }),
);

export type PolicyListReplacement = {
  environmentId: string;
  projectId: string;
  appId: string;
  /** The environment's complete list, in evaluation order. */
  policies: PolicyRow[];
};

/**
 * Replaces whole policy lists. Insert, delete and reorder have no endpoint of
 * their own, so each sends an environment's full list, which the caller already
 * renders. Batched, so one action across both environments emits a single toast.
 *
 * The new list shows when the refetch lands. Do not write `_order` into the
 * collection to show it sooner: live queries order by `_order`, and rewriting it
 * on every row at once duplicated all of them.
 */
export async function replacePolicyLists(
  replacements: PolicyListReplacement[],
  labels: { loading: string; success: string; error: string },
): Promise<void> {
  if (replacements.length === 0) {
    return;
  }
  const promise = Promise.all(
    replacements.map((r) =>
      getUnkeyClient().gateway.setPolicies({
        project: r.projectId,
        app: r.appId,
        environment: r.environmentId,
        policies: r.policies,
      }),
    ),
  );
  toast.promise(promise, {
    loading: labels.loading,
    success: labels.success,
    error: (err) => getErrorToast(err, labels.error),
  });
  try {
    await trackSave(promise);
  } finally {
    // Also on failure: one environment can be written while the other is not.
    for (const r of replacements) {
      queryClient.invalidateQueries({ queryKey: ["policies", r.environmentId] });
    }
  }
}
