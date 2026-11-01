"use client";

import {
  type PolicyListToast,
  type PolicyRow,
  persistPolicyMutations,
  policies,
  policyListLoad,
  rowKey,
} from "@/lib/collections/deploy/policies";
import {
  type QueueStrategy,
  and,
  createLiveQueryCollection,
  eq,
  queueStrategy,
  useLiveQuery,
  usePacedMutations,
} from "@tanstack/react-db";
import { match } from "@unkey/match";
import { useState, useSyncExternalStore } from "react";
import { useAppScope } from "./context";
import { type QueryResult, memoByArgs, subscribeQueryCache } from "./queries";

export type PolicyWrite =
  | { type: "insert"; row: PolicyRow }
  | { type: "update"; row: PolicyRow; patch: Partial<PolicyRow> }
  | { type: "delete"; row: PolicyRow };

// Each write applies at once and resolves when the server has it. A failed
// write rolls back and rejects; the collection has already toasted it.
export type PolicyWrites = Record<
  "add" | "update" | "delete" | "reorder",
  (writes: PolicyWrite[]) => Promise<void>
>;

// One environment's policies of the scope's app, in gateway order. Idle without an environment.
export function usePolicies(environmentId: string | null): QueryResult<PolicyRow[] | undefined> {
  const { projectId, appId } = useAppScope();
  const query = useLiveQuery(
    () => (environmentId === null ? null : policiesQuery(projectId, appId, environmentId)),
    [projectId, appId, environmentId],
  );
  const loadFailed = useSyncExternalStore(
    subscribeQueryCache,
    () => environmentId !== null && policyListLoad(environmentId).isError,
    () => false,
  );
  const isError = query.isError || loadFailed;
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError,
    error: isError ? policies.utils.lastError : null,
    refetch: () =>
      environmentId === null ? Promise.resolve() : policyListLoad(environmentId).clearError(),
  };
}

// The writes share one queue, so they reach the server in the order they were made
export function usePolicyWrites(): PolicyWrites {
  const [strategy] = useState(() => queueStrategy());
  return {
    add: usePolicyWrite(strategy, TOASTS.add),
    update: usePolicyWrite(strategy, TOASTS.update),
    delete: usePolicyWrite(strategy, TOASTS.delete),
    reorder: usePolicyWrite(strategy, TOASTS.reorder),
  };
}

const TOASTS = {
  add: { loading: "Adding policy...", success: "Policy added", error: "Failed to add policy" },
  update: {
    loading: "Updating policy...",
    success: "Policy updated",
    error: "Failed to update policy",
  },
  delete: {
    loading: "Deleting policy...",
    success: "Policy deleted",
    error: "Failed to delete policy",
  },
  reorder: {
    loading: "Reordering policies...",
    success: "Policies reordered",
    error: "Failed to reorder policies",
  },
} satisfies Record<string, PolicyListToast>;

const policiesQuery = memoByArgs((projectId: string, appId: string, environmentId: string) =>
  createLiveQueryCollection((q) =>
    q
      .from({ p: policies })
      .where(({ p }) =>
        and(eq(p.projectId, projectId), eq(p.appId, appId), eq(p.environmentId, environmentId)),
      )
      .orderBy(({ p }) => p._order),
  ),
);

function usePolicyWrite(strategy: QueueStrategy, labels: PolicyListToast) {
  const write = usePacedMutations<PolicyWrite[], PolicyRow>({
    strategy,
    onMutate: applyWrites,
    mutationFn: ({ transaction }) => persistPolicyMutations(transaction.mutations, labels),
  });
  return (writes: PolicyWrite[]) => write(writes).isPersisted.promise.then(() => undefined);
}

function applyWrites(writes: PolicyWrite[]) {
  for (const write of writes) {
    const key = rowKey(write.row.environmentId, write.row.id);
    match(write)
      .with({ type: "insert" }, ({ row }) => policies.insert(row))
      .with({ type: "update" }, ({ patch }) =>
        policies.update(key, (draft) => {
          Object.assign(draft, patch);
        }),
      )
      .with({ type: "delete" }, () => policies.delete(key))
      .exhaustive();
  }
}
