"use client"
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { queryClient, trpcClient } from "./client";
import { z } from "zod";

const schema = z.object({
  id: z.string(),
  name: z.string(),
});

export const ratelimitNamespaces = createCollection(
  queryCollectionOptions({
    schema,
    queryClient,
    queryKey: ["ratelimitNamespaces"],
    retry: 3,
    queryFn: async () => {
      console.info("DB fetching ratelimitNamespaces");
      return await trpcClient.ratelimit.namespace.list.query();
    },
    getKey: (item) => [item.name].join("::"),
    onInsert: async ({ transaction }) => {
      const { changes: newNamespace } = transaction.mutations[0];
      if (!newNamespace.name) {
        throw new Error("Namespace name is required");
      }

      const p = trpcClient.ratelimit.namespace.create.mutate({ name: newNamespace.name });
      toast.promise(p, {
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
      await p;
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];

      const p = trpcClient.ratelimit.namespace.update.name.mutate({
        namespaceId: original.id,
        name: modified.name,
      });
      toast.promise(p, {
        loading: "Updating namespace...",
        success: "Namespace updated",
        error: "Failed to update namespace",
      });
      await p;
    },
    onDelete: async ({ transaction }) => {
      const { original } = transaction.mutations[0];
      const p = trpcClient.ratelimit.namespace.delete.mutate({ namespaceId: original.id });
      toast.promise(p, {
        loading: "Deleting namespace...",
        success: "Namespace deleted",
        error: "Failed to delete namespace",
      });
      await p;
    },
  }),
);
