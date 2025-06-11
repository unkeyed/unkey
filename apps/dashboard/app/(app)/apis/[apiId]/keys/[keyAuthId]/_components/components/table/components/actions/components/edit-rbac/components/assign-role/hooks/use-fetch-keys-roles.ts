"use client";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

export const useFetchKeysRoles = (limit = 50) => {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    trpc.key.update.rbac.roles.query.useInfiniteQuery(
      {
        limit,
      },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
        onError(err) {
          if (err.data?.code === "NOT_FOUND") {
            toast.error("Failed to Load Roles", {
              description:
                "We couldn't find any roles for this workspace. Please try again or contact support@unkey.dev.",
            });
          } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
            toast.error("Server Error", {
              description:
                "We were unable to load roles. Please try again or contact support@unkey.dev",
            });
          } else {
            toast.error("Failed to Load Roles", {
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
