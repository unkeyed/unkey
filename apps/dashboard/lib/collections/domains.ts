"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "./client";

const schema = z.object({
  id: z.string(),
  domain: z.string(),
  type: z.enum(["custom", "wildcard"]),
  projectId: z.string().nullable(),
});

export type Domain = z.infer<typeof schema>;

export const domains = createCollection<Domain>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["domains"],
    retry: 3,
    queryFn: () => trpcClient.deploy.domain.list.query(),

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
