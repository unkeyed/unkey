"use client";

import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

export const useFetchIdentities = (limit = 50) => {
  const trpcUtils = trpc.useUtils();

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    trpc.identity.query.useInfiniteQuery(
      {
        limit,
      },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
        onError(err) {
          if (err.data?.code === "NOT_FOUND") {
            toast.error("Failed to Load Identities", {
              description:
                "We couldn't find any identities for this workspace. Please try again or contact support@unkey.dev.",
            });
          } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
            toast.error("Server Error", {
              description:
                "We were unable to load identities. Please try again or contact support@unkey.dev",
            });
          } else {
            toast.error("Failed to Load Identities", {
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

  const identities = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) => page.identities);
  }, [data?.pages]);

  const loadMore = () => {
    if (!isFetchingNextPage && hasNextPage) {
      fetchNextPage();
    }
  };

  const refresh = () => {
    trpcUtils.identity.query.invalidate();
  };

  return {
    identities,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore,
    refresh,
  };
};
