"use client";
import type { LastExit } from "@/lib/types/deploy";
import { getUnkeyClient } from "@/lib/unkey-client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { type LoadSubsetOptions, createCollection } from "@tanstack/react-db";
import {
  type Deployment as ApiDeployment,
  DeploymentIssueType,
} from "@unkey/api/models/components";
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

const regionSchema = z.object({
  id: z.string(),
  name: z.string(),
  platform: z.string(),
});

const lastExitSchema = z.object({
  restartCount: z.number().int(),
  exitCode: z.number().int().nullable(),
  signal: z.number().int().nullable(),
  reason: z.string().nullable(),
  finishedAt: z.number().nullable(),
  statusReason: z.string().nullable(),
}) satisfies z.ZodType<LastExit>;

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
      region: regionSchema,
      status: z.enum(INSTANCE_STATUSES),
      lastExit: lastExitSchema.nullable(),
    }),
  ),
  desiredInstanceCount: z.number().int(),
  desiredRegions: z.array(z.object({ region: regionSchema })),
  // desiredRegions is the app's current settings, which an older deployment may not match.
  targetRegions: z.array(z.object({ region: regionSchema, replicas: z.number().int() })),
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
  crashLoop: z
    .object({
      reason: z.string().nullable(),
      message: z.string(),
      exitCode: z.number().int().nullable(),
    })
    .nullable(),
  domains: z.array(z.string()),
});

export type Deployment = z.infer<typeof deploymentSchema>;

export function isPrebuilt(deployment: Pick<Deployment, "source">): boolean {
  return deployment.source === "oci";
}

export function hasBuildLogs(deployment: Pick<Deployment, "source" | "status">): boolean {
  return !isPrebuilt(deployment) && deployment.status !== "skipped";
}

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
type RowDetails = Pick<
  Deployment,
  | "projectId"
  | "appId"
  | "environmentId"
  | "hasOpenApiSpec"
  | "desiredState"
  | "instances"
  | "desiredInstanceCount"
  | "desiredRegions"
  | "triggerReason"
> & {
  // A server older than this bundle omits it, as after a rollback with a tab still open
  targetRegions?: Deployment["targetRegions"];
};

// Once a row in one of these statuses has no instances left, nothing changes its
// details, so a refetch keeps the held details instead of asking again
const SETTLED_DETAIL_STATUSES = new Set<DeploymentStatus>(["superseded", "cancelled", "skipped"]);

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
  details: RowDetails,
  projectId: string,
): Deployment {
  const { git, docker, runtime } = deployment;
  const crashLoop = deployment.issues.find((issue) => issue.type === DeploymentIssueType.CrashLoop);
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
    gitCommitAuthorHandle: git?.author?.handle ?? null,
    gitCommitAuthorAvatarUrl: git?.author?.avatarUrl ?? DEFAULT_AVATAR_URL,
    gitCommitTimestamp: git?.commitTimestamp ?? null,
    prNumber: git?.prNumber ?? null,
    forkRepositoryFullName: git?.forkRepository ?? null,
    hasOpenApiSpec: details.hasOpenApiSpec,
    status: deployment.status,
    desiredState: details.desiredState,
    instances: details.instances,
    desiredInstanceCount: details.desiredInstanceCount,
    desiredRegions: details.desiredRegions,
    targetRegions: details.targetRegions ?? [],
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
    crashLoop: crashLoop
      ? {
          reason: crashLoop.meta?.reason ?? null,
          message: crashLoop.message,
          exitCode: crashLoop.meta?.exitCode ?? null,
        }
      : null,
    domains: deployment.domains ?? [],
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
    // Live queries here hold a fixed window of the newest rows. Paged lists
    // use the API cursor outside the collection, so a request past the window
    // has nothing to load
    isPastWindow: opts?.cursor !== undefined || (opts?.offset ?? 0) > 0,
  };
}

type DeploymentSubset = ReturnType<typeof readDeploymentSubset> & { projectId: string };

async function fetchDeploymentRows(subset: DeploymentSubset): Promise<ApiDeployment[]> {
  try {
    if (subset.deploymentId !== undefined) {
      const response = await getUnkeyClient().deployments.getDeployment({
        deploymentId: subset.deploymentId,
      });
      return [response.data];
    }
    const response = await getUnkeyClient().deployments.listDeployments({
      project: subset.projectId,
      app: subset.appId,
      environment:
        subset.appId !== undefined && subset.environmentIds.length === 1
          ? subset.environmentIds[0]
          : undefined,
      status: subset.statuses.length > 0 ? subset.statuses : undefined,
      branch:
        subset.appId !== undefined && subset.branches.length > 0 ? subset.branches : undefined,
      startTime: subset.startTime,
      endTime: subset.endTime,
      limit: Math.min(subset.limit ?? API_PAGE_LIMIT, API_PAGE_LIMIT),
    });
    return response.result.data;
  } catch (error) {
    if (error instanceof NotFoundErrorResponse) {
      return [];
    }
    throw error;
  }
}

/**
 * Merges each API row with its tRPC details. `known` returns a row already
 * held, whose details are reused when nothing can have changed them. Rows whose
 * details belong to another project are dropped.
 */
export async function withDeploymentDetails(
  rows: ApiDeployment[],
  projectId: string,
  known: (id: string) => Deployment | undefined,
): Promise<Deployment[]> {
  if (rows.length === 0) {
    return [];
  }
  const held = new Map(
    rows.flatMap((row) => {
      const previous = known(row.id);
      return previous &&
        previous.status === row.status &&
        previous.instances.length === 0 &&
        SETTLED_DETAIL_STATUSES.has(row.status)
        ? [[row.id, previous] as const]
        : [];
    }),
  );
  const missing = rows.filter((row) => !held.has(row.id)).map((row) => row.id);
  const details = missing.length > 0 ? await loadDeploymentDetails(missing) : {};
  return rows.flatMap((row) => {
    const detail = details[row.id] ?? held.get(row.id);
    return detail?.projectId === projectId ? [toDeployment(row, detail, projectId)] : [];
  });
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
    // Keys start with ["deployments", projectId, appId] so a refetch can name
    // one project or one app
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
            subset.isPastWindow,
          ]
        : ["deployments"];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx): Promise<Deployment[]> => {
      const subset = readDeploymentSubset(ctx.meta?.loadSubsetOptions);
      const { projectId } = subset;
      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }
      if (subset.isPastWindow) {
        return [];
      }
      const rows = await fetchDeploymentRows({ ...subset, projectId });
      return withDeploymentDetails(rows, projectId, (id) => deployments.get(id));
    },
    getKey: (item) => item.id,
    id: "deployments",
  }),
);
