"use client";

import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { IconCodeBranchOutline18 } from "@unkey/icons";
import { useMemo, useState } from "react";
import { useSearchBranches } from "./use-search-branches";

type BranchFieldProps = {
  projectId: string;
  installationId: number;
  repositoryFullName: string;
  branch: string;
  disabled: boolean;
  onChange: (branch: string) => void;
};

export function BranchField({
  projectId,
  installationId,
  repositoryFullName,
  branch,
  disabled,
  onChange,
}: BranchFieldProps) {
  const [owner = "", repo = ""] = repositoryFullName.split("/");
  const [query, setQuery] = useState("");
  const { data: details } = trpc.github.getRepositoryDetails.useQuery(
    { projectId, installationId, owner, repo, defaultBranch: branch },
    { refetchOnWindowFocus: false },
  );
  const { searchResults, isSearching } = useSearchBranches({
    projectId,
    installationId,
    owner,
    repo,
    query,
  });

  const options = useMemo(() => {
    const names = new Set([
      branch,
      ...searchResults.map((b) => b.name),
      ...(details?.branches ?? []).map((b) => b.name),
    ]);
    return [...names].map((name) => ({ label: name, value: name }));
  }, [branch, searchResults, details?.branches]);

  return (
    <FormCombobox
      aria-label="Branch"
      options={options}
      value={branch}
      disabled={disabled}
      onSelect={(value) => {
        setQuery("");
        if (value && value !== branch) {
          onChange(value);
        }
      }}
      onChange={(e) => setQuery(e.currentTarget.value)}
      placeholder={
        <span className="flex items-center gap-1.5">
          <IconCodeBranchOutline18 className="size-3.5 shrink-0 text-gray-11" />
          <span className="truncate">{branch}</span>
        </span>
      }
      searchPlaceholder="Search branches…"
      emptyMessage={isSearching ? "Searching…" : "No branches found."}
    />
  );
}
