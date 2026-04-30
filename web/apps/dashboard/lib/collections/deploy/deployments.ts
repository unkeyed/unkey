"use client";
import { flagCodes } from "@/lib/trpc/routers/deploy/network/utils";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { DEPLOYMENT_STATUSES } from "./deployment-status";
import { validateProjectIdInQuery } from "./utils";

export const deploymentSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  environmentId: z.string(),
  gitCommitSha: z.string().nullable(),
  gitBranch: z.string(),
  gitCommitMessage: z.string().nullable(),
  gitCommitAuthorHandle: z.string().nullable(),
  gitCommitAuthorAvatarUrl: z.string(),
  gitCommitTimestamp: z.number().int().nullable(),
  prNumber: z.number().int().nullable(),
  forkRepositoryFullName: z.string().nullable(),
  hasOpenApiSpec: z.boolean(),
  status: z.enum(DEPLOYMENT_STATUSES),
  instances: z.array(
    z.object({
      id: z.string(),
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
  createdAt: z.number(),
});

export type Deployment = z.infer<typeof deploymentSchema>;

export const DEPLOYMENTS_DEFAULT_LIMIT = 100;

type ParsedFilter = { field: Array<string | number>; operator: string; value?: unknown };

function extractStringFilter(filters: ParsedFilter[], fieldName: string, operator: string) {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "string" ? value : undefined;
}

function extractNumberFilter(filters: ParsedFilter[], fieldName: string, operator: string) {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "number" ? value : undefined;
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
      const { filters } = parseLoadSubsetOptions(opts);
      const projectId = extractStringFilter(filters, "projectId", "eq");
      const startTime = extractNumberFilter(filters, "createdAt", "gte");
      const endTime = extractNumberFilter(filters, "createdAt", "lte");
      return projectId
        ? ["deployments", projectId, startTime ?? null, endTime ?? null]
        : ["deployments"];
    },
    retry: 3,
    syncMode: "on-demand",
    refetchInterval: 5000,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateProjectIdInQuery(options?.where);
      const { filters } = parseLoadSubsetOptions(options);
      const projectId = extractStringFilter(filters, "projectId", "eq");

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      const startTime = extractNumberFilter(filters, "createdAt", "gte");
      const endTime = extractNumberFilter(filters, "createdAt", "lte");

      return trpcClient.deploy.deployment.list.query({
        projectId,
        ...(startTime !== undefined && { startTime }),
        ...(endTime !== undefined && { endTime }),
      });
    },
    getKey: (item) => item.id,
    id: "deployments",
  }),
);
