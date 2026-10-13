"use client";
import { useSyncExternalStore } from "react";
import { queryClient } from "./client";

export type LoadSource = {
  readonly isError: boolean;
  clearError: () => Promise<unknown>;
};

function subscribe(onChange: () => void): () => void {
  return queryClient.getQueryCache().subscribe(onChange);
}

/**
 * useLiveQuery's isError only covers exceptions inside sync, so load failures
 * are read from the collection. A failed load records its error there and
 * emits nothing, and a retry that returns no rows emits nothing either.
 * Rereading `failed` on every query cache event keeps it in step with retries.
 */
export function useCollectionLoad(...sources: LoadSource[]): {
  failed: boolean;
  retry: () => void;
} {
  const failed = useSyncExternalStore(
    subscribe,
    () => sources.some((source) => source.isError),
    () => false,
  );
  const retry = () => {
    for (const source of sources) {
      source.clearError().catch(() => undefined);
    }
  };
  return { failed, retry };
}
