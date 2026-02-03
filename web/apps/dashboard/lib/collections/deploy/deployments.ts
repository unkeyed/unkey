"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

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
    }),
  ),
  cpuMillicores: z.number().int(),
  memoryMib: z.number().int(),
  createdAt: z.number(),
});

export type Deployment = z.infer<typeof schema>;

export function createDeploymentsCollection(projectId: string) {
  if (!projectId) {
    throw new Error("projectId is required to create deployments collection");
  }

  return createCollection<Deployment>(
    queryCollectionOptions({
      queryClient,
      queryKey: [projectId, "deployments"],
      retry: 3,
      refetchInterval: 5000,
      queryFn: () => trpcClient.deploy.deployment.list.query({ projectId }),
      getKey: (item) => item.id,
      id: `${projectId}-deployments`,
    }),
  );
}
