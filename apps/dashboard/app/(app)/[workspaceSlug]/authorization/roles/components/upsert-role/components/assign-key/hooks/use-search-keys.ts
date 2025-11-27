import { useTRPC } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";

import { useQuery } from "@tanstack/react-query";

export const useSearchKeys = (query: string, debounceMs = 300) => {
  const trpc = useTRPC();
  const [debouncedQuery, setDebouncedQuery] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query.trim());
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  const { data, isLoading, error } = useQuery(
    trpc.authorization.roles.keys.search.queryOptions(
      { query: debouncedQuery },
      {
        enabled: debouncedQuery.length > 0, // Only search when there's a debounced query
        staleTime: 30_000,
      },
    ),
  );

  const searchResults = useMemo(() => {
    return data?.keys || [];
  }, [data?.keys]);

  const isSearching = query.trim() !== debouncedQuery || (debouncedQuery.length > 0 && isLoading);

  return {
    searchResults,
    isSearching,
    searchError: error,
  };
};
