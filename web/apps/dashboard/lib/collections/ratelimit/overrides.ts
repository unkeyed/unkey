"use client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  namespaceId: z.string(),
  identifier: z.string(),
  limit: z.number(),
  duration: z.number(),
});
export type RatelimitOverride = z.infer<typeof schema>;

export const ratelimitOverrides = createCollection<RatelimitOverride, string>(
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
      const override = schema.parse(changes);

      const mutation = getUnkeyClient().ratelimit.setOverride({
        namespace: override.namespaceId,
        identifier: override.identifier,
        limit: override.limit,
        duration: override.duration,
      });
      toast.promise(mutation, {
        loading: "Creating override...",
        success: "Override created",
        error: (err) => getErrorToast(err, "Failed to create override"),
      });
      await mutation;
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];

      // setOverride keys on namespace and identifier, not the override id, and
      // the dialog only ever edits limit and duration
      const mutation = getUnkeyClient().ratelimit.setOverride({
        namespace: original.namespaceId,
        identifier: original.identifier,
        limit: modified.limit,
        duration: modified.duration,
      });
      toast.promise(mutation, {
        loading: "Updating override...",
        success: "Override updated",
        error: (err) => getErrorToast(err, "Failed to update override"),
      });
      await mutation;
    },
    onDelete: async ({ transaction }) => {
      const { original } = transaction.mutations[0];
      const mutation = getUnkeyClient().ratelimit.deleteOverride({
        namespace: original.namespaceId,
        identifier: original.identifier,
      });
      toast.promise(mutation, {
        loading: "Deleting override...",
        success: "Override deleted",
        error: (err) => getErrorToast(err, "Failed to delete override"),
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
