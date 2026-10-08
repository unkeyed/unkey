"use client";
import { flagCodes } from "@/lib/trpc/routers/deploy/network/utils";
import { getUnkeyClient } from "@/lib/unkey-client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { type LoadSubsetOptions, createCollection } from "@tanstack/react-db";
import type { Deployment as ApiDeployment } from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import {
  DEPLOYMENT_STATUSES,
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatus,
} from "./deployment-status";
import { INSTANCE_STATUSES } from "./instance-status";
import { type ParsedFilter, extractStringFilter, extractStringValues } from "./utils";

export const deploymentSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  source: z.enum(["unknown", "git", "oci"]),
  requestedImage: z.string().nullable(),
  gitCommitSha: z.string().nullable(),
  gitBranch: z.string(),
  gitCommitMessage: z.string().nullable(),
  gitCommitAuthorHandle: z.string().nullable(),
  gitCommitAuthorAvatarUrl: z.string(),
  gitCommitTimestamp: z.number().int().nullable(),
  prNumber: z.number().int().nullable(),
  forkRepositoryFullName: z.string().nullable(),
  resolvedImage: z.string().nullable(),
  hasOpenApiSpec: z.boolean(),
  status: z.enum(DEPLOYMENT_STATUSES),
  desiredState: z.enum(["running", "stopped"]),
  instances: z.array(
    z.object({
      id: z.string(),
      region: z.object({
        id: z.string(),
        name: z.string(),
        platform: z.string(),
      }),
      flagCode: z.enum(flagCodes),
      status: z.enum(INSTANCE_STATUSES),
    }),
  ),
  desiredInstanceCount: z.number().int(),
  desiredRegions: z.array(
    z.object({
      region: z.object({
        id: z.string(),
        name: z.string(),
        platform: z.string(),
      }),
      flagCode: z.enum(flagCodes),
    }),
  ),
  cpuMillicores: z.number().int(),
  memoryMib: z.number().int(),
  storageMib: z.number().int(),
  port: z.number().int(),
  upstreamProtocol: z.enum(["http1", "h2c"]),
  healthcheck: z
    .object({
      method: z.enum(["GET", "POST"]),
      path: z.string(),
      intervalSeconds: z.number(),
      timeoutSeconds: z.number(),
      failureThreshold: z.number(),
      initialDelaySeconds: z.number(),
    })
    .nullable(),
  shutdownSignal: z.enum(["SIGTERM", "SIGINT", "SIGQUIT", "SIGKILL"]),
  trigger: z.enum(["unknown", "github", "api", "cli", "dashboard", "unkey"]),
  triggeredBy: z.string().nullable(),
  triggerReason: z.string().nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
  // When the build/deploy pipeline finished, from deployment_steps (the only
  // timing stop/wake don't mutate). Null while a build is still in progress or
  // when a deployment has no steps. Powers the row's duration display.
  buildEndedAt: z.number().nullable(),
  // Most-recent exit info across the deployment's instances. Null when
  // no instance has reported a termination yet (healthy deployments).
  // Powers the header "OOMKilled · exit=137" badge — see
  // active-deployment-card/index.tsx:LastExitBadge. Sourced from
  // instances.containerStatus on the trpc layer; the flat shape is kept
  // here so consumers don't have to walk the JSON's optional sub-fields.
  lastExit: z
    .object({
      restartCount: z.number().int(),
      exitCode: z.number().int().nullable(),
      signal: z.number().int().nullable(),
      reason: z.string().nullable(),
      finishedAt: z.number().nullable(),
      statusReason: z.string().nullable(),
    })
    .nullable(),
});

export type Deployment = z.infer<typeof deploymentSchema>;

// The fields the deployment cards, dialogs, and table cells read. Both a
// collection row and a deployments-table row satisfy it.
export type DeploymentSummary = Pick<
  Deployment,
  | "id"
  | "projectId"
  | "appId"
  | "environmentId"
  | "status"
  | "source"
  | "gitCommitSha"
  | "gitBranch"
  | "gitCommitMessage"
  | "gitCommitAuthorHandle"
  | "gitCommitAuthorAvatarUrl"
  | "prNumber"
  | "forkRepositoryFullName"
  | "requestedImage"
  | "resolvedImage"
  | "trigger"
  | "createdAt"
>;

// Every row reads from the public API and merges in what the API does not
// return (parent ids, desired state, runtime details) from one tRPC call. This
// module is the only place that fetches deployments.
const API_PAGE_LIMIT = 100;
const DEFAULT_AVATAR_URL = "https://github.com/identicons/dummy-user.png";

type DeploymentDetailsById = Awaited<
  ReturnType<typeof trpcClient.deploy.deployment.listDetails.query>
>;
type DeploymentDetails = DeploymentDetailsById[string];

// Subsets of one page load resolve within milliseconds of each other, so their
// detail lookups wait a moment and share one request
const DETAILS_BATCH_MS = 10;
let detailsBatch: { ids: Set<string>; result: Promise<DeploymentDetailsById> } | null = null;

function loadDeploymentDetails(ids: string[]): Promise<DeploymentDetailsById> {
  if (!detailsBatch) {
    const batch = { ids: new Set<string>(), result: Promise.resolve({}) };
    batch.result = new Promise((resolve) => setTimeout(resolve, DETAILS_BATCH_MS)).then(
      async () => {
        detailsBatch = null;
        const all = [...batch.ids];
        const chunks: string[][] = [];
        for (let i = 0; i < all.length; i += API_PAGE_LIMIT) {
          chunks.push(all.slice(i, i + API_PAGE_LIMIT));
        }
        const results = await Promise.all(
          chunks.map((deploymentIds) =>
            trpcClient.deploy.deployment.listDetails.query({ deploymentIds }),
          ),
        );
        return Object.assign({}, ...results);
      },
    );
    detailsBatch = batch;
  }
  for (const id of ids) {
    detailsBatch.ids.add(id);
  }
  return detailsBatch.result;
}

function toDeployment(
  deployment: ApiDeployment,
  details: DeploymentDetails,
  projectId: string,
): Deployment {
  const { git, docker, runtime } = deployment;
  return {
    id: deployment.id,
    projectId,
    appId: details.appId,
    environmentId: details.environmentId,
    source: git ? "git" : docker ? "oci" : "unknown",
    requestedImage: docker?.image ?? null,
    resolvedImage: docker?.resolvedImage ?? null,
    gitCommitSha: git?.commitSha ?? null,
    gitBranch: git?.branch ?? "",
    gitCommitMessage: git?.commitMessage ?? null,
    gitCommitAuthorHandle: git?.authorHandle ?? null,
    gitCommitAuthorAvatarUrl: git?.authorAvatarUrl ?? DEFAULT_AVATAR_URL,
    gitCommitTimestamp: git?.commitTimestamp ?? null,
    prNumber: git?.prNumber ?? null,
    forkRepositoryFullName: git?.forkRepositoryFullName ?? null,
    hasOpenApiSpec: details.hasOpenApiSpec,
    status: deployment.status,
    desiredState: details.desiredState,
    instances: details.instances,
    desiredInstanceCount: details.desiredInstanceCount,
    desiredRegions: details.desiredRegions,
    cpuMillicores: Math.round(runtime.vCpus * 1000),
    memoryMib: runtime.memoryMib,
    storageMib: runtime.storageMib,
    port: runtime.port,
    upstreamProtocol: runtime.upstreamProtocol,
    healthcheck: runtime.healthcheck
      ? {
          method: runtime.healthcheck.method,
          path: runtime.healthcheck.path,
          intervalSeconds: runtime.healthcheck.intervalSeconds ?? 0,
          timeoutSeconds: runtime.healthcheck.timeoutSeconds ?? 0,
          failureThreshold: runtime.healthcheck.failureThreshold ?? 0,
          initialDelaySeconds: runtime.healthcheck.initialDelaySeconds ?? 0,
        }
      : null,
    shutdownSignal: runtime.shutdownSignal,
    trigger: deployment.trigger.via,
    triggeredBy: deployment.trigger.actor?.id ?? null,
    triggerReason: details.triggerReason,
    createdAt: deployment.createdAt,
    updatedAt: deployment.updatedAt ?? null,
    buildEndedAt: deployment.finishedAt ?? null,
    lastExit: details.lastExit,
  };
}

function extractNumberFilter(filters: ParsedFilter[], fieldName: string, operator: string) {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "number" ? value : undefined;
}

// The same reading feeds queryKey and queryFn so the two can never disagree
// about which server subset a live query maps to. A live query on a single id
// loads that row on its own, however old it is.
function readDeploymentSubset(opts: LoadSubsetOptions | undefined) {
  const { filters, limit } = parseLoadSubsetOptions(opts);
  const lastKey = opts?.cursor?.lastKey;
  const inclusiveEnd = extractNumberFilter(filters, "createdAt", "lte");
  return {
    limit,
    projectId: extractStringFilter(filters, "projectId"),
    appId: extractStringFilter(filters, "appId"),
    deploymentId: extractStringFilter(filters, "id"),
    statuses: extractStringValues(filters, "status").filter((s): s is DeploymentStatus =>
      Object.hasOwn(DEPLOYMENT_STATUS_LABELS, s),
    ),
    environmentIds: extractStringValues(filters, "environmentId"),
    branches: extractStringValues(filters, "gitBranch"),
    startTime: extractNumberFilter(filters, "createdAt", "gte"),
    endTime:
      extractNumberFilter(filters, "createdAt", "lt") ??
      (inclusiveEnd === undefined ? undefined : inclusiveEnd + 1),
    cursor: typeof lastKey === "string" ? lastKey : undefined,
    offset: opts?.offset !== undefined && opts.offset > 0 ? opts.offset : undefined,
  };
}

type DeploymentSubset = ReturnType<typeof readDeploymentSubset> & { projectId: string };

// The filters of a subset without its page window, so every page of one list
// shares an entry
function listKey(subset: DeploymentSubset): string {
  return JSON.stringify([
    subset.projectId,
    subset.appId,
    subset.statuses,
    subset.environmentIds,
    subset.branches,
    subset.startTime,
    subset.endTime,
  ]);
}

// Lists whose last page came back shorter than asked. A window that grows past
// such a list, as useLiveInfiniteQuery does right after a short first page, has
// nothing left to load
const exhaustedLists = new Set<string>();

// A grown window (load more) can arrive as an offset instead of a cursor. The
// API pages by id, so the cursor is the row this tab already holds at that
// offset, in the same newest-first order the live queries use
function cursorAtOffset(subset: DeploymentSubset, offset: number): string | undefined {
  const held = [...deployments.values()]
    .filter(
      (d) =>
        d.projectId === subset.projectId &&
        (subset.appId === undefined || d.appId === subset.appId) &&
        (subset.statuses.length === 0 || subset.statuses.includes(d.status)) &&
        (subset.environmentIds.length === 0 || subset.environmentIds.includes(d.environmentId)) &&
        (subset.branches.length === 0 || subset.branches.includes(d.gitBranch)) &&
        (subset.startTime === undefined || d.createdAt >= subset.startTime) &&
        (subset.endTime === undefined || d.createdAt < subset.endTime),
    )
    .sort((a, b) => b.createdAt - a.createdAt);
  return held[offset - 1]?.id;
}

async function fetchDeploymentRows(subset: DeploymentSubset): Promise<ApiDeployment[]> {
  if (subset.deploymentId !== undefined) {
    try {
      const response = await getUnkeyClient().deployments.getDeployment({
        deploymentId: subset.deploymentId,
      });
      return [response.data];
    } catch (error) {
      if (error instanceof NotFoundErrorResponse) {
        return [];
      }
      throw error;
    }
  }

  const isNextPage = subset.cursor !== undefined || subset.offset !== undefined;
  if (isNextPage && exhaustedLists.has(listKey(subset))) {
    return [];
  }
  const cursor =
    subset.cursor ??
    (subset.offset === undefined ? undefined : cursorAtOffset(subset, subset.offset));
  if (subset.offset !== undefined && cursor === undefined) {
    return [];
  }
  // The API cursor includes the row it names, so a cursor page asks for one
  // extra row and drops it
  const limit = Math.min((subset.limit ?? API_PAGE_LIMIT) + (cursor ? 1 : 0), API_PAGE_LIMIT);
  const response = await getUnkeyClient().deployments.listDeployments({
    project: subset.projectId,
    app: subset.appId,
    environment:
      subset.appId !== undefined && subset.environmentIds.length === 1
        ? subset.environmentIds[0]
        : undefined,
    status: subset.statuses.length > 0 ? subset.statuses : undefined,
    branch: subset.appId !== undefined && subset.branches.length > 0 ? subset.branches : undefined,
    startTime: subset.startTime,
    endTime: subset.endTime,
    limit,
    cursor,
  });
  if (response.result.pagination.hasMore) {
    exhaustedLists.delete(listKey(subset));
  } else {
    exhaustedLists.add(listKey(subset));
  }
  return response.result.data.filter((d) => d.id !== cursor);
}

/**
 * Global deployments collection.
 *
 * IMPORTANT: All queries MUST filter by projectId:
 * .where(({ deployment }) => eq(deployment.projectId, projectId))
 */
export const deployments = createCollection<Deployment, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const subset = readDeploymentSubset(opts);
      return subset.projectId
        ? [
            "deployments",
            subset.projectId,
            subset.appId ?? null,
            subset.deploymentId ?? null,
            subset.statuses.join(",") || null,
            subset.environmentIds.join(",") || null,
            subset.branches.join(",") || null,
            subset.startTime ?? null,
            subset.endTime ?? null,
            subset.limit ?? null,
            subset.offset ?? null,
            subset.cursor ?? null,
          ]
        : ["deployments"];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      const subset = readDeploymentSubset(ctx.meta?.loadSubsetOptions);
      const { projectId } = subset;
      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      const rows = await fetchDeploymentRows({ ...subset, projectId });
      if (rows.length === 0) {
        return [];
      }
      const details = await loadDeploymentDetails(rows.map((d) => d.id));
      return rows.flatMap((row) => {
        const detail = details[row.id];
        return detail?.projectId === projectId ? [toDeployment(row, detail, projectId)] : [];
      });
    },
    getKey: (item) => item.id,
    id: "deployments",
  }),
);
