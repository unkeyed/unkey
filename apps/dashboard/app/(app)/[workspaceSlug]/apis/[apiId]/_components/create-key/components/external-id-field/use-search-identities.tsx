import { useTRPC } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";

import { useQuery } from "@tanstack/react-query";

export const useSearchIdentities = (query: string, debounceMs = 300) => {
  const trpc = useTRPC();
  const [debouncedQuery, setDebouncedQuery] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query.trim());
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  const { data, isLoading, error } = useQuery(
    trpc.identity.search.queryOptions(
      { query: debouncedQuery },
      {
        enabled: debouncedQuery.length > 0,
        staleTime: 30_000,
      },
    ),
  );

  const searchResults = useMemo(() => {
    return data?.identities || [];
  }, [data?.identities]);

  const isSearching = query.trim() !== debouncedQuery || (debouncedQuery.length > 0 && isLoading);

  return {
    searchResults,
    isSearching,
    searchError: error,
  };
};
