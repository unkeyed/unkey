"use client";

import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import {
  type QueryCollectionUtils,
  parseLoadSubsetOptions,
  queryCollectionOptions,
} from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { queryClient } from "../client";
import type { LoadSource } from "../use-collection-load";
import { ENVIRONMENT_KINDS, type EnvironmentKind } from "./environments";
import { trackSave } from "./pending-redeploy";
import { type Policy, type PolicyInput, fromWirePolicy } from "./policies.schema";
import { extractStringFilter } from "./utils";

/** A Policy plus the identifiers every gateway call is scoped by. */
export type PolicyRow = Policy & {
  environmentId: string;
  projectId: string;
  appId: string;
  _order?: number;
};

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

/**
 * Gateway policies collection. It holds one row for each (environment, policy).
 *
 * IMPORTANT: All queries MUST filter by projectId, appId, and environmentId.
 * `listPolicies` reads one environment, so each environment is its own subset
 * and its own request. The two environments of a page load in parallel.
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
    onInsert: readOnly,
    onUpdate: readOnly,
    onDelete: readOnly,
  }),
);

function readOnly(): never {
  throw new Error("The policies collection is read only. Write through writePolicies.");
}

export type PolicyListReplacement = {
  environmentId: string;
  projectId: string;
  appId: string;
  /** The environment's complete list, in evaluation order. */
  policies: PolicyInput[];
};

export type PolicyListToast = { loading: string; success: string; error: string };

// Settles once the refetch lands. Writing `_order` into the collection to show
// the list sooner duplicated every row in the live query.
export async function replacePolicyLists(
  replacements: PolicyListReplacement[],
  labels: PolicyListToast,
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
    // "all" so a queued edit still reads fresh lists after the page unmounted.
    await Promise.all(
      replacements.map((r) =>
        queryClient.invalidateQueries({
          queryKey: ["policies", r.environmentId],
          refetchType: "all",
        }),
      ),
    );
  }
}

export type PolicyLists = Record<EnvironmentKind, PolicyRow[]>;

export type PolicyEnvironmentIds = Record<EnvironmentKind, string>;

export type PolicyScope = {
  projectId: string;
  appId: string;
  environments: Record<EnvironmentKind, string | null>;
};

export type PolicyEditResult =
  | { type: "write"; lists: Partial<Record<EnvironmentKind, PolicyInput[]>> }
  | { type: "reject"; message: string };

export type PolicyEdit = (lists: PolicyLists) => PolicyEditResult;

type PolicyRead = { type: "lists"; lists: PolicyLists } | { type: "reject"; message: string };

type PolicyWriterIo = {
  read: (environments: PolicyEnvironmentIds) => PolicyRead;
  write: (replacements: PolicyListReplacement[], labels: PolicyListToast) => Promise<void>;
  notify: (message: string) => void;
};

const ENVIRONMENT_MISSING = "Couldn't find the environment. Reload the page and try again.";
const LISTS_OUT_OF_DATE = "Policies are out of date. Reload and try again.";

function loadedEnvironments({
  production,
  preview,
}: PolicyScope["environments"]): PolicyEnvironmentIds | null {
  return production !== null && preview !== null ? { production, preview } : null;
}

export function createPolicyWriter(io: PolicyWriterIo) {
  let chain: Promise<unknown> = Promise.resolve();
  return (scope: PolicyScope, edit: PolicyEdit, labels: PolicyListToast): Promise<boolean> => {
    const run = chain.then(async () => {
      const environments = loadedEnvironments(scope.environments);
      if (!environments) {
        io.notify(ENVIRONMENT_MISSING);
        return false;
      }
      const read = io.read(environments);
      if (read.type === "reject") {
        io.notify(read.message);
        return false;
      }
      const result = edit(read.lists);
      if (result.type === "reject") {
        io.notify(result.message);
        return false;
      }
      const replacements = ENVIRONMENT_KINDS.flatMap((env) => {
        const policies = result.lists[env];
        return policies
          ? [
              {
                environmentId: environments[env],
                projectId: scope.projectId,
                appId: scope.appId,
                policies,
              },
            ]
          : [];
      });
      if (replacements.length === 0) {
        return true;
      }
      try {
        await io.write(replacements, labels);
        return true;
      } catch {
        return false;
      }
    });
    chain = run.catch(() => undefined);
    return run;
  };
}

// The collection syncs from the query cache on a later tick, so the cache is
// the only source that is current right after `replacePolicyLists` settles.
function readCachedLists(environments: PolicyEnvironmentIds): PolicyRead {
  const production = queryClient.getQueryState<PolicyRow[]>(["policies", environments.production]);
  const preview = queryClient.getQueryState<PolicyRow[]>(["policies", environments.preview]);
  if (!production?.data || !preview?.data) {
    return { type: "reject", message: ENVIRONMENT_MISSING };
  }
  if ([production, preview].some((s) => s.isInvalidated || s.status === "error")) {
    return { type: "reject", message: LISTS_OUT_OF_DATE };
  }
  return { type: "lists", lists: { production: production.data, preview: preview.data } };
}

export const writePolicies = createPolicyWriter({
  read: readCachedLists,
  write: replacePolicyLists,
  notify: (message) => toast.error(message),
});
