"use client";

import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import {
  type QueryCollectionUtils,
  parseLoadSubsetOptions,
  queryCollectionOptions,
} from "@tanstack/query-db-collection";
import { type PendingMutation, createCollection } from "@tanstack/react-db";
import { match } from "@unkey/match";
import { toast } from "@unkey/ui";
import { queryClient } from "../client";
import type { LoadSource } from "../use-collection-load";
import { trackSave } from "./pending-redeploy";
import {
  POLICY_ERRORS,
  type Policy,
  type PolicyInput,
  fromWirePolicy,
  policyMatchKey,
} from "./policies.schema";
import { extractStringFilter } from "./utils";

/** A Policy plus the identifiers every gateway call is scoped by. */
export type PolicyRow = Policy & {
  environmentId: string;
  projectId: string;
  appId: string;
  _order: number;
};

type PolicyListScope = Pick<PolicyRow, "environmentId" | "projectId" | "appId">;

export const rowKey = (environmentId: string, policyId: string) => `${environmentId}::${policyId}`;

/**
 * Each environment is its own subset, and any subset that loads clears the
 * collection's isError, so a failure is read from the environment's query.
 * Clearing the collection's error refetches every subset.
 */
export function policyListLoad(environmentId: string): LoadSource {
  return {
    get isError() {
      return queryClient.getQueryState(["policies", environmentId])?.status === "error";
    },
    clearError: () => policies.utils.clearError(),
  };
}

async function listPolicyRows({
  projectId,
  appId,
  environmentId,
}: PolicyListScope): Promise<PolicyRow[]> {
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
}

/**
 * Gateway policies collection. It holds one row for each (environment, policy).
 *
 * IMPORTANT: All queries MUST filter by projectId, appId, and environmentId.
 * `listPolicies` reads one environment, so each environment is its own subset
 * and its own request. The two environments of a page load in parallel.
 *
 * Writes run in a serial queue of transactions whose mutationFn is
 * `persistPolicyMutations`.
 */
export const policies = createCollection<
  PolicyRow,
  string,
  QueryCollectionUtils<PolicyRow, string>
>(
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

      return listPolicyRows({ projectId, appId, environmentId });
    },
    getKey: (row) => rowKey(row.environmentId, row.id),
    id: "policies",
  }),
);

export function policyList(environmentId: string): PolicyRow[] {
  return policies.toArray
    .filter((row) => row.environmentId === environmentId)
    .sort((a, b) => a._order - b._order);
}

export function toPolicyInput(row: PolicyRow): PolicyInput {
  const { id: _id, environmentId: _env, projectId: _project, appId: _app, _order, ...input } = row;
  return input;
}

export type PolicyListToast = { loading: string; success: string; error: string };

export type PolicyMutation = Pick<
  PendingMutation<PolicyRow>,
  "type" | "original" | "modified" | "changes"
>;

/**
 * Finds a row in a fresh list. A list replace gives every policy a new id, so a
 * write queued behind one falls back to the type and name, when they are unique.
 */
export function resolveRow(rows: readonly PolicyRow[], row: PolicyRow): PolicyRow | null {
  const byId = rows.find((r) => r.id === row.id);
  if (byId) {
    return byId;
  }
  const key = policyMatchKey(row.type, row.name);
  const named = rows.filter((r) => policyMatchKey(r.type, r.name) === key);
  return named.length === 1 ? named[0] : null;
}

function mutationTarget(m: PolicyMutation): PolicyRow {
  return m.type === "update" && "id" in m.original ? { ...m.modified, ...m.original } : m.modified;
}

export function nextPolicyList(fresh: readonly PolicyRow[], mutations: PolicyMutation[]) {
  const inserted: PolicyRow[] = [];
  let rows = [...fresh];
  for (const m of mutations) {
    if (m.type === "insert") {
      const key = policyMatchKey(m.modified.type, m.modified.name);
      if ([...rows, ...inserted].some((r) => policyMatchKey(r.type, r.name) === key)) {
        throw new Error(POLICY_ERRORS.sharedName);
      }
      inserted.push(m.modified);
      continue;
    }
    const target = resolveRow(rows, mutationTarget(m));
    if (!target) {
      throw new Error(POLICY_ERRORS.gone);
    }
    rows =
      m.type === "delete"
        ? rows.filter((r) => r !== target)
        : rows.map((r) => (r === target ? { ...m.modified, id: target.id } : r));
  }
  return [...rows.sort((a, b) => a._order - b._order), ...inserted];
}

function variantBody(policy: PolicyInput) {
  return match(policy)
    .with({ type: "keyauth" }, (p) => ({ keyauth: p.keyauth }))
    .with({ type: "ratelimit" }, (p) => ({ ratelimit: p.ratelimit }))
    .with({ type: "firewall" }, (p) => ({ firewall: p.firewall }))
    .with({ type: "openapi" }, (p) => ({ openapi: p.openapi }))
    .with({ type: "logging" }, (p) => ({ logging: p.logging }))
    .exhaustive();
}

function updateBody(m: PolicyMutation) {
  const changed = Object.keys(m.changes);
  if (changed.length === 1 && changed[0] === "enabled") {
    return { enabled: m.modified.enabled };
  }
  const policy = toPolicyInput(m.modified);
  return {
    name: policy.name,
    match: policy.match?.length ? policy.match : null,
    ...variantBody(policy),
  };
}

function isInPlace(mutations: PolicyMutation[]): boolean {
  return mutations.every((m) => m.type === "update" && !("_order" in m.changes));
}

async function updateInPlace(mutations: PolicyMutation[]) {
  for (const m of mutations) {
    const { environmentId, projectId, appId } = m.modified;
    const cached = queryClient.getQueryData<PolicyRow[]>(["policies", environmentId]);
    const target = cached ? resolveRow(cached, mutationTarget(m)) : m.modified;
    if (!target) {
      throw new Error(POLICY_ERRORS.gone);
    }
    await getUnkeyClient().gateway.updatePolicy({
      project: projectId,
      app: appId,
      environment: environmentId,
      policyId: target.id,
      ...updateBody(m),
    });
  }
}

// Insert, delete and reorder have no endpoint of their own. Reading the list
// first keeps a replace from writing back a list that another write changed.
async function replaceList(scope: PolicyListScope, mutations: PolicyMutation[]) {
  const fresh = await listPolicyRows(scope);
  const next = nextPolicyList(fresh, mutations);
  await getUnkeyClient().gateway.setPolicies({
    project: scope.projectId,
    app: scope.appId,
    environment: scope.environmentId,
    policies: next.map(toPolicyInput),
  });
}

export async function persistPolicyMutations(
  mutations: PolicyMutation[],
  labels: PolicyListToast,
): Promise<void> {
  const byEnvironment = Map.groupBy(mutations, (m) => m.modified.environmentId);
  const promise = Promise.all(
    [...byEnvironment.values()].map((envMutations) =>
      isInPlace(envMutations)
        ? updateInPlace(envMutations)
        : replaceList(envMutations[0].modified, envMutations),
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
    await Promise.all(
      [...byEnvironment.keys()].map((environmentId) =>
        queryClient.invalidateQueries({ queryKey: ["policies", environmentId] }),
      ),
    );
  }
}
