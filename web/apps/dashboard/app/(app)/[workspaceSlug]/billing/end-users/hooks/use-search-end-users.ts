import { trpc } from "@/lib/trpc/client";
import { useMemo, useState } from "react";

export function useSearchIdentities(initialQuery: string = "") {
  const [searchValue, setSearchValue] = useState(initialQuery);

  const trimmedSearchValue = searchValue.trim();

  // Use the same identity search endpoint as create key
  const { data, isLoading, isFetching } = trpc.identity.search.useQuery(
    { query: trimmedSearchValue },
    {
      enabled: trimmedSearchValue.length > 0,
      staleTime: 30 * 1000, // Cache results for 30 seconds
    },
  );

  const searchResults = useMemo(() => {
    return data?.identities ?? [];
  }, [data]);

  const isSearching = isLoading || isFetching;

  return {
    searchValue,
    setSearchValue,
    searchResults,
    isSearching,
    trimmedSearchValue,
  };
}

// Re-export as useSearchEndUsers for backward compatibility
export { useSearchIdentities as useSearchEndUsers };