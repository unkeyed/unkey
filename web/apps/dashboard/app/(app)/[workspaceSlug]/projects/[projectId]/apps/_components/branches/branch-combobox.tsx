"use client";

import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { useMemo, useState } from "react";
import { useBranchOptions } from "./use-branch-options";

export type BranchConnection = {
  repositoryFullName: string;
  installationId: number;
  branch: string;
};

type BranchComboboxProps = {
  projectId: string;
  connection: BranchConnection;
  value: string;
  onSelect: (branch: string) => void;
  label?: string;
  "aria-label"?: string;
  className?: string;
};

export function BranchCombobox({
  projectId,
  connection,
  value,
  onSelect,
  ...props
}: BranchComboboxProps) {
  const options = useBranchItems(projectId, connection, value);

  return (
    <FormCombobox
      {...props}
      options={options.items}
      value={value}
      onSelect={(selected) => {
        if (selected) {
          onSelect(selected);
        }
      }}
      onQueryChange={options.setQuery}
      loading={options.isLoading}
      searchPlaceholder="Search branches..."
      emptyMessage={options.isSearching ? "Searching..." : "No branches found."}
    />
  );
}

function useBranchItems(projectId: string, connection: BranchConnection, branch: string) {
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
