"use client";

import { useIdentities } from "@/lib/identities-query";
import { getErrorMessage } from "@/lib/unkey-client";
import { toast } from "@unkey/ui";

export const useFetchIdentities = () => {
  const { identities, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useIdentities({
    onError: (error) => {
      toast.error("Failed to Load Identities", {
        description: getErrorMessage(error, "We were unable to load identities. Please try again."),
      });
    },
  });

  return {
    identities,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore: () => fetchNextPage(),
  };
};
