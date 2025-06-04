import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

export const useSearchPermissions = (query: string) => {
  const { data, isLoading, error } = trpc.authorization.permissions.search.useQuery(
    { query },
    {
      enabled: query.trim().length > 0, // Only search when there's a query
      staleTime: 10_000,
    },
  );

  const searchResults = useMemo(() => {
    return data?.permissions || [];
  }, [data?.permissions]);

  return {
    searchResults,
    isSearching: isLoading,
    searchError: error,
  };
};
