import { useIdentities } from "@/lib/identities-query";
import { toast } from "@unkey/ui";
import { useDeferredValue, useEffect } from "react";

export const useSearchIdentities = (query: string) => {
  const search = query.trim();
  const deferredSearch = useDeferredValue(search);
  const { identities, isLoading, isError } = useIdentities({
    search: deferredSearch,
    enabled: deferredSearch.length > 0,
  });

  useEffect(() => {
    if (isError) {
      toast.error("Failed to Search Identities", {
        description: "We were unable to search identities. Please try again.",
      });
    }
  }, [isError]);

  return {
    searchResults: identities,
    isSearching: search !== deferredSearch || (deferredSearch.length > 0 && isLoading),
  };
};
