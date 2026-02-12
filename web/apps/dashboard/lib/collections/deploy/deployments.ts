"use client";
import { flagCodes } from "@/lib/trpc/routers/deploy/network/utils";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { parseProjectIdFromWhere, validateProjectIdInQuery } from "./utils";

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  environmentId: z.string(),
  gitCommitSha: z.string().nullable(),
  gitBranch: z.string(),
  gitCommitMessage: z.string().nullable(),
  gitCommitAuthorHandle: z.string().nullable(),
  gitCommitAuthorAvatarUrl: z.string(),
  gitCommitTimestamp: z.number().int().nullable(),
  // OpenAPI
  hasOpenApiSpec: z.boolean(),
  // Deployment status
  status: z.enum(["pending", "building", "deploying", "network", "ready", "failed"]),
  instances: z.array(
    z.object({
      id: z.string(),
      region: z.string(),
      flagCode: z.enum(flagCodes),
    }),
  ),
  cpuMillicores: z.number().int(),
  memoryMib: z.number().int(),
  createdAt: z.number(),
});

export type Deployment = z.infer<typeof schema>;

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
      const projectId = parseProjectIdFromWhere(opts.where);
      return projectId ? ["deployments", projectId] : ["deployments"];
    },
    retry: 3,
    syncMode: "on-demand",
    refetchInterval: 5000,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateProjectIdInQuery(options?.where);
      const projectId = parseProjectIdFromWhere(options?.where);

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      return trpcClient.deploy.deployment.list.query({ projectId });
    },
    getKey: (item) => item.id,
    id: "deployments",
  }),
);
