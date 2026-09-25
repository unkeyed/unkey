"use client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useInfiniteQuery } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useMemo } from "react";

// No need to fetch more than 10 items, because combobox allows seeing 6 items at a time so even if users scroll 10 items are more than enough.
export const MAX_PERMS_FETCH_LIMIT = 10;

export const keysRbacPermissionsQueryOptions = (limit = MAX_PERMS_FETCH_LIMIT) => ({
  queryKey: ["keys-rbac-permissions", limit] as const,
  queryFn: ({ pageParam }: { pageParam?: string }) =>
    getUnkeyClient().permissions.listPermissions({ cursor: pageParam, limit }),
});

export const useFetchPermissions = (limit = MAX_PERMS_FETCH_LIMIT) => {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = useInfiniteQuery({
    ...keysRbacPermissionsQueryOptions(limit),
    getNextPageParam: (lastPage) =>
      lastPage.result.pagination.hasMore ? lastPage.result.pagination.cursor : undefined,
    onError(err: unknown) {
      const { message, description } = getErrorToast(err, "Failed to Load Permissions");
      toast.error(message, { description });
    },
  });

  const permissions = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) => page.result.data);
  }, [data?.pages]);

  const loadMore = () => {
    if (!isFetchingNextPage && hasNextPage) {
      fetchNextPage();
    }
  };

  return {
    permissions,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore,
  };
};
