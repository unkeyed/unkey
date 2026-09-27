"use client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import type { RatelimitOverride as ApiRatelimitOverride } from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient } from "../client";
import { extractStringFilter } from "../deploy/utils";

const schema = z.object({
  id: z.string(),
  namespaceId: z.string(),
  identifier: z.string(),
  limit: z.number(),
  duration: z.number(),
});
export type RatelimitOverride = z.infer<typeof schema>;

/**
 * Ratelimit overrides collection.
 *
 * IMPORTANT: All queries MUST filter by namespaceId:
 * .where(({ override }) => eq(override.namespaceId, namespaceId))
 *
 * Add eq(override.identifier, identifier) to load one override instead of the
 * whole namespace. Mutations need their row loaded, so a component that updates
 * or deletes an override holds such a query, see `useOverride`.
 */
export const ratelimitOverrides = createCollection<RatelimitOverride, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const namespaceId = extractStringFilter(filters, "namespaceId");
      const identifier = extractStringFilter(filters, "identifier");
      return namespaceId
        ? ["ratelimitOverrides", namespaceId, identifier ?? null]
        : ["ratelimitOverrides"];
    },
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const namespaceId = extractStringFilter(filters, "namespaceId");
      if (!namespaceId) {
        throw new Error("Query must include eq(collection.namespaceId, namespaceId) constraint");
      }

      const identifier = extractStringFilter(filters, "identifier");
      if (identifier !== undefined) {
        try {
          const { data } = await getUnkeyClient().ratelimit.getOverride({
            namespace: namespaceId,
            identifier,
          });
          return data.identifier === identifier ? [toOverride(namespaceId, data)] : [];
        } catch (error) {
          if (error instanceof NotFoundErrorResponse) {
            return [];
          }
          throw error;
        }
      }

      const pages = await getUnkeyClient().ratelimit.listOverrides({
        namespace: namespaceId,
        limit: 100,
      });
      const all: RatelimitOverride[] = [];
      for await (const page of pages) {
        for (const o of page.result.data) {
          all.push(toOverride(namespaceId, o));
        }
      }
      return all;
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

function toOverride(namespaceId: string, o: ApiRatelimitOverride): RatelimitOverride {
  return {
    id: o.overrideId,
    namespaceId,
    identifier: o.identifier,
    limit: o.limit,
    duration: o.duration,
  };
}
