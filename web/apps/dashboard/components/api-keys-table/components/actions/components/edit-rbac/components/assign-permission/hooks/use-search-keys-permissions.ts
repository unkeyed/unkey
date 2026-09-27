import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";

export const useSearchPermissions = (query: string, debounceMs = 300) => {
  const [debouncedQuery, setDebouncedQuery] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query.trim());
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  const { data, isLoading, error } = useQuery({
    queryKey: ["keys-rbac-permissions-search", debouncedQuery],
    queryFn: () => getUnkeyClient().permissions.listPermissions({ search: debouncedQuery }),
    enabled: debouncedQuery.length > 0,
    staleTime: 30_000,
  });

  const searchResults = useMemo(() => data?.result.data ?? [], [data?.result.data]);

  const isSearching = query.trim() !== debouncedQuery || (debouncedQuery.length > 0 && isLoading);

  return {
    searchResults,
    isSearching,
    searchError: error,
  };
};
