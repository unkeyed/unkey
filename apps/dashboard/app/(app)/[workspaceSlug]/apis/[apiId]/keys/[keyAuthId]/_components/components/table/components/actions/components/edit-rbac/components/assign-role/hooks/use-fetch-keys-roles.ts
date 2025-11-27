"use client";
import { useTRPC } from "@/lib/trpc/client";
import { useInfiniteQuery } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useMemo } from "react";

// No need to fetch more than 10 items, because combobox allows seeing 6 items at a time so even if users scroll 10 items are more than enough.
export const MAX_ROLES_FETCH_LIMIT = 10;

export const useFetchKeysRoles = (limit = MAX_ROLES_FETCH_LIMIT) => {
  const trpc = useTRPC();
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, error } =
    useInfiniteQuery(
      trpc.key.update.rbac.roles.query.infiniteQueryOptions(
        {
          limit,
        },
        {
          getNextPageParam: (lastPage) => lastPage.nextCursor,
        },
      ),
    );

  if (error) {
    if (error.data?.code === "NOT_FOUND") {
      toast.error("Failed to Load Roles", {
        description:
          "We couldn't find any roles for this workspace. Please try again or contact support@unkey.dev.",
      });
    } else if (error.data?.code === "INTERNAL_SERVER_ERROR") {
      toast.error("Server Error", {
        description: "We were unable to load roles. Please try again or contact support@unkey.dev",
      });
    } else {
      toast.error("Failed to Load Roles", {
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

  const roles = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) => page.roles);
  }, [data?.pages]);

  const loadMore = () => {
    if (!isFetchingNextPage && hasNextPage) {
      fetchNextPage();
    }
  };

  return {
    roles,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore,
  };
};
