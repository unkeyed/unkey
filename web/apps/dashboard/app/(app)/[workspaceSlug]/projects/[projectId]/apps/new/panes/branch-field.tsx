"use client";

import { useBuildSource } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/hooks/use-build-source";
import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { useMemo, useState } from "react";
import { useSearchBranches } from "./repository/use-search-branches";

type Connection = {
  repositoryFullName: string;
  installationId: number;
  branch: string;
};

type BranchFieldProps = { value: string | null; onChange: (branch: string) => void };

export function BranchField(props: BranchFieldProps) {
  const { projectId, connection } = useBuildSource();
  return connection ? (
    <BranchCombobox projectId={projectId} connection={connection} {...props} />
  ) : null;
}

function BranchCombobox({
  projectId,
  connection,
  value,
  onChange,
}: BranchFieldProps & { projectId: string; connection: Connection }) {
  const branch = value ?? connection.branch;
  const options = useBranchOptions(projectId, connection, branch);

  return (
    <FormCombobox
      label="Branch"
      options={options.items}
      value={branch}
      onSelect={(selected) => {
        options.setQuery("");
        if (selected) {
          onChange(selected);
        }
      }}
      onChange={(e) => options.setQuery(e.currentTarget.value)}
      loading={options.isLoading}
      searchPlaceholder="Search branches..."
      emptyMessage={options.isSearching ? "Searching..." : "No branches found."}
    />
  );
}

function useBranchOptions(projectId: string, connection: Connection, branch: string) {
  const [owner = "", repo = ""] = connection.repositoryFullName.split("/");
  const [query, setQuery] = useState("");
  const details = trpc.github.getRepositoryDetails.useQuery(
    {
      projectId,
      installationId: connection.installationId,
      owner,
      repo,
      defaultBranch: connection.branch,
    },
    { refetchOnWindowFocus: false },
  );
  const { searchResults, isSearching } = useSearchBranches({
    projectId,
    installationId: connection.installationId,
    owner,
    repo,
    query,
  });
  const repositoryDefault = details.data?.repositoryDefaultBranch;

  const items = useMemo(() => {
    const names = new Set([
      branch,
      ...(details.data?.branches ?? []).map((b) => b.name),
      ...searchResults.map((b) => b.name),
    ]);
    return [...names].map((name) => ({
      value: name,
      selectedLabel: name,
      label:
        name === repositoryDefault ? (
          <span className="flex w-full items-center justify-between gap-2">
            <span className="truncate">{name}</span>
            <span className="shrink-0 text-xs text-gray-9">default</span>
          </span>
        ) : (
          name
        ),
    }));
  }, [branch, details.data?.branches, searchResults, repositoryDefault]);

  return { items, setQuery, isSearching, isLoading: details.isLoading };
}
