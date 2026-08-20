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
 * An edit uses `gateway.updatePolicy`. Insert, delete, and reorder have no
 * endpoint, so they send the whole list to `gateway.setPolicies`.
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
      const mutations = transaction.mutations.map((m) => dispatchUpdate(m.modified));
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

    onInsert: async ({ transaction }) => {
      const all = dispatchSetForChanges(
        transaction.mutations.map((m) => ({ key: m.key, row: m.modified, removed: false })),
      );
      const plural = transaction.mutations.length > 1;
      toast.promise(all, {
        loading: plural ? "Adding policies..." : "Adding policy...",
        success: plural ? "Policies added" : "Policy added",
        error: (err) =>
          getErrorToast(err, plural ? "Failed to add policies" : "Failed to add policy"),
      });
      await trackSave(all);
    },

    onDelete: async ({ transaction }) => {
      const all = dispatchSetForChanges(
        transaction.mutations.map((m) => ({ key: m.key, row: m.original, removed: true })),
      );
      const plural = transaction.mutations.length > 1;
      toast.promise(all, {
        loading: plural ? "Deleting policies..." : "Deleting policy...",
        success: plural ? "Policies deleted" : "Policy deleted",
        error: (err) =>
          getErrorToast(err, plural ? "Failed to delete policies" : "Failed to delete policy"),
      });
      await trackSave(all);
    },
  }),
);

/**
 * Whole-list reorder. Batched so one drag across both environments emits a
 * single toast. Each entry carries that environment's rows already in the
 * desired order, so no policy has to be matched by name or id.
 */
export async function reorderPolicies(
  reorders: { environmentId: string; projectId: string; appId: string; policies: PolicyRow[] }[],
): Promise<void> {
  if (reorders.length === 0) {
    return;
  }
  const promise = Promise.all(
    reorders.map(async (r) => {
      await getUnkeyClient().gateway.setPolicies({
        project: r.projectId,
        app: r.appId,
        environment: r.environmentId,
        policies: r.policies,
      });
    }),
  );
  toast.promise(promise, {
    loading: "Reordering policies...",
    success: "Policies reordered",
    error: (err) => getErrorToast(err, "Failed to reorder policies"),
  });
  await trackSave(promise);
  for (const r of reorders) {
    queryClient.invalidateQueries({ queryKey: ["policies", r.environmentId] });
  }
}

/** Next `_order` in an environment, so an optimistic insert lands at the end. */
export function nextPolicyOrder(environmentId: string): number {
  const orders = rowsForEnvironment(policies.state, environmentId).map((r) => r._order ?? -1);
  return Math.max(-1, ...orders) + 1;
}

export function findPolicyByName(environmentId: string, name: string): PolicyRow | undefined {
  return rowsForEnvironment(policies.state, environmentId).find((r) => r.name === name);
}

/**
 * Rows in evaluation order. Row-only fields ride along: the SDK's outbound
 * schema drops every key the API does not declare.
 */
export function orderedPolicies(rows: PolicyRow[]): PolicyRow[] {
  return [...rows].sort((a, b) => (a._order ?? 0) - (b._order ?? 0));
}

export function environmentsInMutations(
  rows: PolicyRow[],
): { environmentId: string; projectId: string; appId: string }[] {
  const byEnvironment = new Map<
    string,
    { environmentId: string; projectId: string; appId: string }
  >();
  for (const row of rows) {
    byEnvironment.set(row.environmentId, {
      environmentId: row.environmentId,
      projectId: row.projectId,
      appId: row.appId,
    });
  }
  return Array.from(byEnvironment.values());
}

function rowsForEnvironment(
  state: ReadonlyMap<string, PolicyRow>,
  environmentId: string,
): PolicyRow[] {
  const rows: PolicyRow[] = [];
  for (const [key, row] of state) {
    if (key.startsWith(`${environmentId}::`)) {
      rows.push(row);
    }
  }
  return rows;
}

// Callers resolve `row` themselves: only they know whether the mutation
// carries it as `modified` or `original`.
export type PolicyChange = { key: string; row: PolicyRow; removed: boolean };

/**
 * The environment's full policy list with `changes` folded in.
 *
 * A mutation handler runs before TanStack DB recomputes optimistic state, so
 * `policies.state` still holds pre-mutation rows. Since a full replace sends
 * the whole list, reading state alone would wipe an environment on first
 * insert and undo a delete.
 */
export function policiesAfterMutations(
  state: ReadonlyMap<string, PolicyRow>,
  environmentId: string,
  changes: PolicyChange[],
): PolicyRow[] {
  const byKey = new Map<string, PolicyRow>();
  for (const row of rowsForEnvironment(state, environmentId)) {
    byKey.set(rowKey(row.environmentId, row.id), row);
  }

  for (const change of changes) {
    if (change.row.environmentId !== environmentId) {
      continue;
    }
    if (change.removed) {
      byKey.delete(change.key);
    } else {
      byKey.set(change.key, change.row);
    }
  }

  return orderedPolicies(Array.from(byKey.values()));
}

function dispatchSetForChanges(changes: PolicyChange[]): Promise<unknown[]> {
  return Promise.all(
    environmentsInMutations(changes.map((c) => c.row)).map(({ environmentId, projectId, appId }) =>
      getUnkeyClient().gateway.setPolicies({
        project: projectId,
        app: appId,
        environment: environmentId,
        policies: policiesAfterMutations(policies.state, environmentId, changes),
      }),
    ),
  );
}

export function dispatchUpdate(row: PolicyRow): Promise<unknown> {
  return getUnkeyClient().gateway.updatePolicy({
    ...row,
    project: row.projectId,
    app: row.appId,
    environment: row.environmentId,
    policyId: row.id,
  });
}
