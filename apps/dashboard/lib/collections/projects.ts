"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "./client";

const schema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  gitRepositoryUrl: z.string().nullable(),
  updatedAt: z.number().int().nullable()
});


type Schema = z.infer<typeof schema>;

export const projects = createCollection<Schema>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["projects"],
    retry: 3,
    queryFn: async () => {
      console.info("DB fetching projects");
      return await trpcClient.deploy.project.list.query();
    },
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      const { changes: newNamespace } = transaction.mutations[0];

      const p = trpcClient.deploy.project.create.mutate(schema.parse({
        id: "created", // will be replaced by the actual ID after creation
        name: newNamespace.name,
        slug: newNamespace.slug,
        gitRepositoryUrl: newNamespace.gitRepositoryUrl ?? null,
        updatedAt: null,
      }))
      toast.promise(p, {
        loading: "Creating project...",
        success: "Project created",
        error: (res) => {
          console.error("Failed to create project", res);
          return {
            message: "Failed to create project",
            description: res.message,
          };
        },
      });
      await p;
    },
    // onDelete: async ({ transaction }) => {
    //   const { original } = transaction.mutations[0];
    //   const p = trpcClient.deploy.project.delete.mutate({ projectId: original.id });
    //   toast.promise(p, {
    //     loading: "Deleting project...",
    //     success: "Project deleted",
    //     error: "Failed to delete project",
    //   });
    //   await p;
    // },
  }),
);
