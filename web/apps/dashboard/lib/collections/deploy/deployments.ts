"use client";
import { flagCodes } from "@/lib/trpc/routers/deploy/network/utils";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { DEPLOYMENT_STATUSES, type DeploymentStatus } from "./deployment-status";
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

export const DEPLOYMENTS_DEFAULT_LIMIT = 100;

function extractNumberFilter(filters: ParsedFilter[], fieldName: string, operator: string) {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "number" ? value : undefined;
}

function extractDeploymentIds(filters: ParsedFilter[]): string[] | undefined {
  const id = extractStringFilter(filters, "id");
  if (id !== undefined) {
    return [id];
  }
  const ids = filters.find((f) => f.field.at(-1) === "id" && f.operator === "in")?.value;
  return Array.isArray(ids)
    ? ids.filter((v): v is string => typeof v === "string").sort()
    : undefined;
}

// The same reading feeds queryKey and queryFn so the two can never disagree
// about which server subset a live query maps to. A live query on ids (an app's
// current deployment, a request log's deployment) loads those rows on their own,
// however old they are; the paged /deployments list does not go through
// the collection at all.
function readDeploymentSubset(opts: Parameters<typeof parseLoadSubsetOptions>[0]) {
  const { filters, limit } = parseLoadSubsetOptions(opts);
  return {
    limit,
    projectIds: extractStringValues(filters, "projectId"),
    statuses: extractStringValues(filters, "status").filter((s): s is DeploymentStatus =>
      (DEPLOYMENT_STATUSES as readonly string[]).includes(s),
    ),
    appId: extractStringFilter(filters, "appId"),
    deploymentIds: extractDeploymentIds(filters),
    startTime: extractNumberFilter(filters, "createdAt", "gte"),
    endTime: extractNumberFilter(filters, "createdAt", "lte"),
  };
}

/**
 * Global deployments collection.
 *
 * IMPORTANT: All queries MUST filter by projectId with eq or inArray:
 * .where(({ deployment }) => eq(deployment.projectId, projectId))

 */
export const deployments = createCollection<Deployment, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      if (opts.cursor) {
        return ["deployments", "next-page"];
      }
      const subset = readDeploymentSubset(opts);
      return subset.projectIds.length > 0
        ? [
            "deployments",
            subset.projectIds.join(","),
            subset.appId ?? null,
            subset.startTime ?? null,
            subset.endTime ?? null,
            subset.statuses.join(",") || null,
            subset.limit ?? null,
            subset.deploymentIds?.join(",") ?? null,
          ]
        : ["deployments"];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      if (ctx.meta?.loadSubsetOptions?.cursor) {
        return [];
      }
      const { projectIds, appId, deploymentIds, statuses, startTime, endTime, limit } =
        readDeploymentSubset(ctx.meta?.loadSubsetOptions);

      if (projectIds.length === 0) {
        throw new Error("Query must include an eq or inArray constraint on collection.projectId");
      }
      if (deploymentIds?.length === 0) {
        return [];
      }

      const perProject = await Promise.all(
        projectIds.map((projectId) =>
          trpcClient.deploy.deployment.list.query({
            projectId,
            ...(appId !== undefined && { appId }),
            ...(deploymentIds !== undefined && { deploymentIds }),
            ...(statuses.length > 0 && { statuses }),
            ...(startTime !== undefined && { startTime }),
            ...(endTime !== undefined && { endTime }),
            ...(limit !== undefined && { limit }),
          }),
        ),
      );
      return perProject.flatMap((result) => result.deployments);
    },
    getKey: (item) => item.id,
    id: "deployments",
  }),
);
