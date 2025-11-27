"use client";
import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useMemo } from "react";

import { useInfiniteQuery } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

const MAX_IDENTITY_FETCH_LIMIT = 10;
export const useFetchIdentities = (limit = MAX_IDENTITY_FETCH_LIMIT) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, error } =
    useInfiniteQuery(
      trpc.identity.query.infiniteQueryOptions(
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
      toast.error("Failed to Load Identities", {
        description:
          "We couldn't find any identities for this workspace. Please try again or contact support@unkey.dev.",
      });
    } else if (error.data?.code === "INTERNAL_SERVER_ERROR") {
      toast.error("Server Error", {
        description:
          "We were unable to load identities. Please try again or contact support@unkey.dev",
      });
    } else {
      toast.error("Failed to Load Identities", {
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
    queryClient.invalidateQueries(trpc.identity.query.pathFilter());
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
