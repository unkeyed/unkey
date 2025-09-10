"use client"
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { queryClient, trpcClient } from "./client";
import { z } from "zod";

const schema = z.object({
  namespaceId: z.string(),
  identifier: z.string(),
  limit: z.number(),
  duration: z.number(),
});

export const ratelimitOverrides = createCollection<Schema>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["ratelimitOverrides"],
    queryFn: async () => {
      console.info("DB fetching ratelimitOverrides");
      return await trpcClient.ratelimit.override.list.query();
    },
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      const { changes } = transaction.mutations[0];

      const mutation = trpcClient.ratelimit.override.create.mutate(schema.parse(changes));
      toast.promise(mutation, {
        loading: "Creating override...",
        success: "Override created",
        error: (res) => {
          console.error("Failed to create override", res);
          return {
            message: "Failed to create override",
            description: res.message,
          };
        },
      });
      await mutation;
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];
      const mutation = trpcClient.ratelimit.override.update.mutate({
        id: original.id,
        limit: modified.limit,
        duration: modified.duration,
      });
      toast.promise(mutation, {
        loading: "Updating override...",
        success: "Override updated",
        error: "Failed to update override",
      });
      await mutation;
    },
    onDelete: async ({ transaction }) => {
      const { original } = transaction.mutations[0];
      const mutation = trpcClient.ratelimit.override.delete.mutate({ id: original.id });
      toast.promise(mutation, {
        loading: "Deleting override...",
        success: "Override deleted",
        error: "Failed to delete override",
      });
      await mutation;
    },
  }),
);

ratelimitOverrides.createIndex((row) => [row.namespaceId, row.identifier], {
  name: "unique_identifier_per_namespace",
  options: {
    unique: true,
  },
});
