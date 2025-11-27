"use client";;
import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useMemo } from "react";

import { useInfiniteQuery } from "@tanstack/react-query";

// No need to fetch more than 10 items, because combobox allows seeing 6 items at a time so even if users scroll 10 items are more than enough.
export const MAX_PERMS_FETCH_LIMIT = 10;

export const useFetchPermissions = (limit = MAX_PERMS_FETCH_LIMIT) => {
  const trpc = useTRPC();
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, error } =
    useInfiniteQuery(trpc.authorization.roles.permissions.query.infiniteQueryOptions(
      {
        limit,
      },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
      },
    ));


  if (error) {
    if (error.data?.code === "NOT_FOUND") {
      toast.error("Failed to Load Permissions", {
        description:
          "We couldn't find any permissions for this workspace. Please try again or contact support@unkey.dev.",
      });
    } else if (error.data?.code === "INTERNAL_SERVER_ERROR") {
      toast.error("Server Error", {
        description:
          "We were unable to load permissions. Please try again or contact support@unkey.dev",
      });
    } else {
      toast.error("Failed to Load Permissions", {
        description:
          error.message ||
          "An unexpected error occurred. Please try again or contact support@unkey.dev",
        action: {
          label: "Contact Support",
          onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
        },
      });
    }
  }

  const permissions = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) => page.permissions);
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
