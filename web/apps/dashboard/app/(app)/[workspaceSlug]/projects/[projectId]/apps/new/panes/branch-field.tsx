"use client";

import { useBranchOptions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/branches/use-branch-options";
import { useBuildSource } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/hooks/use-build-source";
import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { useMemo, useState } from "react";

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
  const options = useBranchItems(projectId, connection, branch);

  return (
    <FormCombobox
      label="Branch"
      options={options.items}
      value={branch}
      onSelect={(selected) => {
        if (selected) {
          onChange(selected);
        }
      }}
      onQueryChange={options.setQuery}
      loading={options.isLoading}
      searchPlaceholder="Search branches..."
      emptyMessage={options.isSearching ? "Searching..." : "No branches found."}
    />
  );
}

function useBranchItems(projectId: string, connection: Connection, branch: string) {
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
  const { options, isSearching } = useBranchOptions({
    source: { projectId, installationId: connection.installationId, owner, repo },
    branches: details.data?.branches ?? [],
    pinned: branch,
    query,
  });
  const repositoryDefault = details.data?.repositoryDefaultBranch;

  const items = useMemo(
    () =>
      options.map(({ name }) => ({
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
      })),
    [options, repositoryDefault],
  );

  return { items, setQuery, isSearching, isLoading: details.isLoading };
}
