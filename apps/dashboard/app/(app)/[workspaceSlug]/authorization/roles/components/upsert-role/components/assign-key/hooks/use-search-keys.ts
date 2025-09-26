import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";

export const useSearchKeys = (query: string, debounceMs = 300) => {
  const [debouncedQuery, setDebouncedQuery] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query.trim());
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  const { data, isLoading, error } = trpc.authorization.roles.keys.search.useQuery(
    { query: debouncedQuery },
    {
      enabled: debouncedQuery.length > 0, // Only search when there's a debounced query
      staleTime: 30_000,
    },
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
