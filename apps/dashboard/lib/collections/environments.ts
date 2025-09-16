"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "./client";

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  slug: z.string(),
});

export type Environment = z.infer<typeof schema>;

export const environments = createCollection<Environment>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["environments"],
    retry: 3,
    queryFn: () => trpcClient.deploy.environment.list.query(),
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
    onUpdate: async () => {
      throw new Error("Not implemented");
      //  const { changes: updatedNamespace } = transaction.mutations[0];
      //
      //  const p = trpcClient.deploy.project.update.mutate(schema.parse({
      //    id: updatedNamespace.id,
      //    name: updatedNamespace.name,
      //    slug: updatedNamespace.slug,
      //    gitRepositoryUrl: updatedNamespace.gitRepositoryUrl ?? null,
      //    updatedAt: new Date(),
      //  }));
      //  toast.promise(p, {
      //    loading: "Updating project...",
      //    success: "Project updated",
      //    error: "Failed to update project",
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
