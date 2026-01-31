import { trpc } from "@/lib/trpc/client";
import { useMemo, useState } from "react";

export function useSearchEndUsers(initialQuery: string = "") {
  const [searchValue, setSearchValue] = useState(initialQuery);

  const trimmedSearchValue = searchValue.trim();

  const { data, isLoading, isFetching } = trpc.customerBilling.endUsers.search.useQuery(
    { query: trimmedSearchValue },
    {
      enabled: trimmedSearchValue.length > 0,
      staleTime: 30 * 1000, // Cache results for 30 seconds
    },
  );

  const searchResults = useMemo(() => {
    return data?.endUsers ?? [];
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