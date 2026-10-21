import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";

export type BranchSource = {
  projectId: string;
  installationId: number;
  owner: string;
  repo: string;
};

export function useSearchBranches(source: BranchSource | null, query: string, debounceMs = 300) {
  const [debouncedQuery, setDebouncedQuery] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query.trim());
    }, debounceMs);
    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  const { data, isInitialLoading } = trpc.github.searchBranches.useQuery(
    {
      projectId: source?.projectId ?? "",
      installationId: source?.installationId ?? 0,
      owner: source?.owner ?? "",
      repo: source?.repo ?? "",
      query: debouncedQuery,
    },
    {
      enabled: source !== null && debouncedQuery.length > 0,
      staleTime: 30_000,
    },
  );

  const searchResults = useMemo(() => data?.branches ?? [], [data?.branches]);
  const isSearching = source !== null && (query.trim() !== debouncedQuery || isInitialLoading);

  return { searchResults, isSearching };
}
