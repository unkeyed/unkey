"use client";

import type { QueryClient } from "@tanstack/react-query";

/**
 * Clear React Query cache specifically
 */
export function clearQueryCache(queryClient: QueryClient) {
  queryClient.clear();
  queryClient.getQueryCache().clear();
  queryClient.getMutationCache().clear();
}

/**
 * Invalidate and refetch all queries
 * Useful when you want to refresh all data without clearing the cache entirely
 */
export async function refetchAllQueries(queryClient: QueryClient) {
  await queryClient.invalidateQueries();
  await queryClient.refetchQueries();
}
