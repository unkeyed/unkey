import { useMemo } from "react";
import { type BranchSource, useSearchBranches } from "./use-search-branches";

export type BranchOption = { name: string; lastPushDate: string | null };

export function useBranchOptions({
  source,
  branches,
  pinned,
  query,
}: {
  source: BranchSource | null;
  branches: readonly BranchOption[];
  pinned: string;
  query: string;
}) {
  const { searchResults, isSearching } = useSearchBranches(source, query);

  const options = useMemo(() => {
    const byName = new Map<string, BranchOption>();
    const candidates: BranchOption[] = [
      branches.find((branch) => branch.name === pinned) ?? { name: pinned, lastPushDate: null },
      ...branches,
      ...searchResults.map((branch) => ({ name: branch.name, lastPushDate: null })),
    ];
    for (const branch of candidates) {
      if (branch.name && !byName.has(branch.name)) {
        byName.set(branch.name, branch);
      }
    }
    return [...byName.values()];
  }, [branches, pinned, searchResults]);

  return { options, isSearching };
}
