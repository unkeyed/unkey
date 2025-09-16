"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "./client";

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  environmentId: z.string(),
  // Git information
  // TEMP: Git fields as non-nullable for UI development with mock data
  // TODO: Convert to nullable (.nullable()) when real git integration is added
  // In production, deployments may not have git metadata if triggered manually
  gitCommitSha: z.string(),
  gitBranch: z.string(),
  gitCommitMessage: z.string(),
  gitCommitAuthorName: z.string(),
  gitCommitAuthorEmail: z.string(),
  gitCommitAuthorUsername: z.string(),
  gitCommitAuthorAvatarUrl: z.string(),
  gitCommitTimestamp: z.number().int(),
  // Immutable configuration snapshot
  runtimeConfig: z.object({
    regions: z.array(
      z.object({
        region: z.string(),
        vmCount: z.number().min(1).max(100),
      }),
    ),
    cpus: z.number().min(1).max(16),
    memory: z.number().min(1).max(1024),
  }),

  // Deployment status
  status: z.enum(["pending", "building", "deploying", "network", "ready", "failed"]),
  createdAt: z.number(),
});

export type Deployment = z.infer<typeof schema>;

export const deployments = createCollection<Deployment>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["deployments"],
    retry: 3,
    queryFn: () => trpcClient.deploy.deployment.list.query(),
    getKey: (item) => item.id,
    onInsert: async () => {
      throw new Error("Not implemented");
    },
    onDelete: async () => {
      throw new Error("Not implemented");
    },
  }),
);
