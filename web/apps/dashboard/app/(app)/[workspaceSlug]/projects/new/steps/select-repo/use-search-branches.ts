import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";

type SearchBranchesParams = {
  projectId: string;
  installationId: number;
  owner: string;
  repo: string;
  query: string;
  debounceMs?: number;
};

export function useSearchBranches({
  projectId,
  installationId,
  owner,
  repo,
  query,
  debounceMs = 300,
}: SearchBranchesParams) {
  const [debouncedQuery, setDebouncedQuery] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query.trim());
    }, debounceMs);
    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  const { data, isLoading } = trpc.github.searchBranches.useQuery(
    { projectId, installationId, owner, repo, query: debouncedQuery },
    {
      enabled: debouncedQuery.length > 0,
      staleTime: 30_000,
    },
  );

  const searchResults = useMemo(() => data?.branches ?? [], [data?.branches]);
  const isSearching = query.trim() !== debouncedQuery || (debouncedQuery.length > 0 && isLoading);

  return { searchResults, isSearching };
}
