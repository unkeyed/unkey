import { useIdentities } from "@/lib/identities-query";
import { getErrorMessage } from "@/lib/unkey-client";
import { toast } from "@unkey/ui";
import { useDeferredValue } from "react";

export const useSearchIdentities = (query: string) => {
  const search = query.trim();
  const deferredSearch = useDeferredValue(search);
  const { identities, isLoading } = useIdentities({
    search: deferredSearch,
    enabled: deferredSearch.length > 0,
    onError: (error) => {
      toast.error("Failed to Search Identities", {
        description: getErrorMessage(
          error,
          "We were unable to search identities. Please try again.",
        ),
      });
    },
  });

  return {
    searchResults: identities,
    isSearching: search !== deferredSearch || (deferredSearch.length > 0 && isLoading),
  };
};
