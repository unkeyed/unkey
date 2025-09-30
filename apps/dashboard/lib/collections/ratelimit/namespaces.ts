"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  name: z.string().min(1).max(50),
});

export type RatelimitNamespace = z.infer<typeof schema>;

export const ratelimitNamespaces = createCollection<RatelimitNamespace>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["ratelimitNamespaces"],
    retry: 3,
    queryFn: async () => {
      return await trpcClient.ratelimit.namespace.list.query();
    },
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      const { changes: newNamespace } = transaction.mutations[0];
      if (!newNamespace.name) {
        throw new Error("Namespace name is required");
      }

      const mutation = trpcClient.ratelimit.namespace.create.mutate({
        name: newNamespace.name,
      });
      toast.promise(mutation, {
        loading: "Creating namespace...",
        success: "Namespace created",
        error: (res) => {
          console.error("Failed to create namespace", res);
          return {
            message: "Failed to create namespace",
            description: res.message,
          };
        },
      });
      await mutation;
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];

      const mutation = trpcClient.ratelimit.namespace.update.name.mutate({
        namespaceId: original.id,
        name: modified.name,
      });
      toast.promise(mutation, {
        loading: "Updating namespace...",
        success: "Namespace updated",
        error: "Failed to update namespace",
      });
      await mutation;
    },
    onDelete: async ({ transaction }) => {
      const { original } = transaction.mutations[0];
      const mutation = trpcClient.ratelimit.namespace.delete.mutate({
        namespaceId: original.id,
      });
      toast.promise(mutation, {
        loading: "Deleting namespace...",
        success: "Namespace deleted",
        error: "Failed to delete namespace",
      });
      await mutation;
    },
  }),
);

ratelimitNamespaces.createIndex((row) => row.name, {
  name: "name",
  options: {
    unique: true,
  },
});
