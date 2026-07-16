"use client";

import { useIdentities } from "@/lib/identities-query";
import { toast } from "@unkey/ui";
import { useEffect } from "react";

export const useFetchIdentities = () => {
  const { identities, isLoading, isError, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useIdentities();

  useEffect(() => {
    if (isError) {
      toast.error("Failed to Load Identities", {
        description: "We were unable to load identities. Please try again.",
      });
    }
  }, [isError]);

  return {
    identities,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore: () => fetchNextPage(),
  };
};
