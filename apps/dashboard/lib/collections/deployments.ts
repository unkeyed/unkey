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
  gitCommitSha: z.string().nullable(),
  gitBranch: z.string().nullable(),
  gitCommitMessage: z.string().nullable(),
  gitCommitAuthorName: z.string().nullable(),
  gitCommitAuthorEmail: z.string().nullable(),
  gitCommitAuthorUsername: z.string().nullable(),
  gitCommitAuthorAvatarUrl: z.string().nullable(),
  gitCommitTimestamp: z.number().int().nullable(),

  // Immutable configuration snapshot
  runtimeConfig: z.object({
    regions: z.array(z.object({
      region: z.string(),
      vmCount: z.number().min(1).max(100)
    })),
    cpus: z.number().min(1).max(16),
    memory: z.number().min(1).max(1024)
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
    queryFn: () => trpcClient.deployment.list.query(),

    getKey: (item) => item.id,
    onInsert: async () => {
      throw new Error("Not implemented");
      //  const { changes: newNamespace } = transaction.mutations[0];
      //
      //  const p = trpcClient.deploy.project.create.mutate(schema.parse({
      //    id: "created", // will be replaced by the actual ID after creation
      //    name: newNamespace.name,
      //    slug: newNamespace.slug,
      //    gitRepositoryUrl: newNamespace.gitRepositoryUrl ?? null,
      //    updatedAt: null,
      //  }))
      //  toast.promise(p, {
      //    loading: "Creating project...",
      //    success: "Project created",
      //    error: (res) => {
      //      console.error("Failed to create project", res);
      //      return {
      //        message: "Failed to create project",
      //        description: res.message,
      //      };
      //    },
      //  });
      //  await p;
    },
    onDelete: async () => {
      throw new Error("Not implemented");
      //   const { original } = transaction.mutations[0];
      //   const p = trpcClient.deploy.project.delete.mutate({ projectId: original.id });
      //   toast.promise(p, {
      //     loading: "Deleting project...",
      //     success: "Project deleted",
      //     error: "Failed to delete project",
      //   });
      //   await p;
    },
  }),
);
