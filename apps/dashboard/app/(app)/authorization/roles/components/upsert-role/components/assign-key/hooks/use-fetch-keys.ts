"use client";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useMemo } from "react";

// No need to fetch more than 10 items, because combobox allows seeing 6 items at a time so even if users scroll 10 items are more than enough.
export const MAX_KEYS_FETCH_LIMIT = 10;

export const useFetchKeys = (limit = MAX_KEYS_FETCH_LIMIT) => {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    trpc.authorization.roles.keys.query.useInfiniteQuery(
      {
        limit,
      },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
        onError(err) {
          if (err.data?.code === "NOT_FOUND") {
            toast.error("Failed to Load Keys", {
              description:
                "We couldn't find any keys for this workspace. Please try again or contact support@unkey.dev.",
            });
          } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
            toast.error("Server Error", {
              description:
                "We were unable to load keys. Please try again or contact support@unkey.dev",
            });
          } else {
            toast.error("Failed to Load Keys", {
              description:
                err.message ||
                "An unexpected error occurred. Please try again or contact support@unkey.dev",
              action: {
                label: "Contact Support",
                onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
              },
            });
          }
        },
      },
    );

  const keys = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) => page.keys);
  }, [data?.pages]);

  const loadMore = () => {
    if (!isFetchingNextPage && hasNextPage) {
      fetchNextPage();
    }
  };

  return {
    keys,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore,
  };
};
