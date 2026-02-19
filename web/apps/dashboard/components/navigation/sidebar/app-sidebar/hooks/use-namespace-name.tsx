"use client";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

/**
 * Hook to fetch namespace name by ID.
 * Returns the namespace name if found.
 */
export const useNamespaceName = (namespaceId?: string) => {
  // Fetch all namespaces to find the one we need
  const { data } = trpc.ratelimit.namespace.list.useQuery(undefined, {
    enabled: !!namespaceId,
  });

  const namespaceName = useMemo(() => {
    if (!data || !namespaceId) {
      return undefined;
    }
    const namespace = data.find((n) => n.id === namespaceId);
    return namespace?.name;
  }, [data, namespaceId]);

  return namespaceName;
};
