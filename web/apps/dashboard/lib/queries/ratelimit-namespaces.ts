"use client";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";

const NAMESPACE_PAGE_SIZE = 50;

export function useNamespaces({ search }: { search: string }) {
  const query = useInfiniteQuery({
    queryKey: queryKeys.ratelimit.namespaces.list(search),
    queryFn: async ({ pageParam, signal }) => {
      const cursor = typeof pageParam === "string" ? pageParam : undefined;
      const response = await getUnkeyClient().ratelimit.listNamespaces(
        { limit: NAMESPACE_PAGE_SIZE, cursor, search: search || undefined },
        { signal },
      );
      const { data, pagination } = response.result;
      if (pagination.hasMore && !pagination.cursor) {
        throw new Error("Namespace API returned a continuation page without a cursor");
      }
      return { namespaces: data, cursor: pagination.hasMore ? pagination.cursor : undefined };
    },
    getNextPageParam: (lastPage) => lastPage.cursor,
    keepPreviousData: true,
  });

  return {
    ...query,
    namespaces: query.data?.pages.flatMap((page) => page.namespaces) ?? [],
  };
}

// No single-namespace endpoint exists, and search matches ids as well as names
export function useNamespace(namespaceId: string) {
  return useQuery({
    queryKey: queryKeys.ratelimit.namespaces.detail(namespaceId),
    queryFn: async ({ signal }) => {
      const response = await getUnkeyClient().ratelimit.listNamespaces(
        { search: namespaceId, limit: 1 },
        { signal },
      );
      return response.result.data.find((namespace) => namespace.id === namespaceId) ?? null;
    },
  });
}

// The logdrains and root key pickers filter on the client, so they need every namespace
export function useEveryNamespace({ enabled = true }: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: queryKeys.ratelimit.namespaces.every,
    queryFn: async ({ signal }) => {
      const pages = await getUnkeyClient().ratelimit.listNamespaces({ limit: 100 }, { signal });
      return (await Array.fromAsync(pages)).flatMap((page) => page.result.data);
    },
    enabled,
  });
}

export function useInvalidateNamespaces() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: queryKeys.ratelimit.namespaces.all });
}
