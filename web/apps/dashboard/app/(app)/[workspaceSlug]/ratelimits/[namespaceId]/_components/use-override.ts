"use client";
import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";

/**
 * Loads one override into the collection, so the dialogs can update or delete
 * it. Idle without an identifier, as when creating a new override.
 */
export function useOverride(namespaceId: string, identifier: string | undefined) {
  return useLiveQuery(
    (q) =>
      identifier
        ? q
            .from({ override: collection.ratelimitOverrides })
            .where(({ override }) =>
              and(eq(override.namespaceId, namespaceId), eq(override.identifier, identifier)),
            )
            .findOne()
        : null,
    [namespaceId, identifier],
  );
}
