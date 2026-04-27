import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { Check, CodeBranch, Magnifier } from "@unkey/icons";
import { Button, TimestampInfo } from "@unkey/ui";
import { useMemo, useState } from "react";
import { LanguageIcon } from "./language-icon";
import { useSearchBranches } from "./use-search-branches";

export type RepoItem = {
  id: number;
  fullName: string;
  installationId: number;
  defaultBranch: string;
  pushedAt: string | null;
  language: string | null;
};

export const RepoListItem = ({
  repo,
  projectId,
  onSelect,
  disabled,
  loading,
}: {
  repo: RepoItem;
  projectId: string;
  onSelect: (repo: RepoItem, selectedBranch: string) => void;
  disabled: boolean;
  loading: boolean;
}) => {
  const [owner, repoName] = repo.fullName.split("/");
  const [selectedBranch, setSelectedBranch] = useState(repo.defaultBranch);
  const [searchValue, setSearchValue] = useState("");

  const { data: details } = trpc.github.getRepositoryDetails.useQuery(
    {
      projectId,
      installationId: repo.installationId,
      owner,
      repo: repoName,
      defaultBranch: repo.defaultBranch,
    },
    {
      refetchOnWindowFocus: false,
    },
  );

  const { searchResults, isSearching } = useSearchBranches({
    projectId,
    installationId: repo.installationId,
    owner,
    repo: repoName,
    query: searchValue,
  });

  const branchOptions = useMemo(() => {
    const preloaded = details?.branches ?? [];
    const seen = new Set<string>();
    const merged: Array<{ name: string }> = [];

    for (const b of searchResults) {
      if (!seen.has(b.name)) {
        seen.add(b.name);
        merged.push(b);
      }
    }
    for (const b of preloaded) {
      if (!seen.has(b.name)) {
        seen.add(b.name);
        merged.push(b);
      }
    }

    for (const name of [selectedBranch, repo.defaultBranch]) {
      if (name && !seen.has(name)) {
        seen.add(name);
        merged.unshift({ name });
      }
    }

    return merged.map((b) => ({ label: b.name, value: b.name }));
  }, [details?.branches, searchResults, selectedBranch, repo.defaultBranch]);

  const isLoading = details === undefined;

  return (
    <div className="flex px-4 py-5 items-center">
      <LanguageIcon language={repo.language} />
      <div className="flex flex-col gap-1 w-40">
        <div className="font-medium text-[13px] text-gray-12 leading-4 truncate max-w-40">
          {repoName}
        </div>
        <div className="font-medium text-[13px] text-gray-10 leading-3">
          <TimestampInfo value={repo.pushedAt ?? 0} displayType="relative" />{" "}
        </div>
      </div>
      <div className="flex gap-2 items-center ml-auto shrink-0">
        {isLoading ? (
          <div className="h-4 w-28 bg-grayA-3 rounded animate-pulse" />
        ) : details.hasDockerfile ? (
          <>
            <Check className="text-success-9" iconSize="sm-regular" />
            <span className="text-gray-11 text-xs">Dockerfile detected</span>
          </>
        ) : (
          <span className="text-gray-11 text-xs">No Dockerfile</span>
        )}
      </div>
      <div className="flex gap-2 items-center">
        <div className="ml-6 w-[200px]">
          {isLoading ? (
            <div className="h-8 w-full bg-grayA-3 rounded-lg animate-pulse" />
          ) : (
            <Combobox
              options={branchOptions}
              value={selectedBranch}
              onSelect={(value) => {
                setSelectedBranch(value);
                setSearchValue("");
              }}
              onChange={(e) => setSearchValue(e.currentTarget.value)}
              placeholder={
                <span className="flex items-center gap-1.5 text-gray-9 text-[13px]">
                  <CodeBranch className="size-3 shrink-0" iconSize="sm-regular" />
                  <span className="truncate">{repo.defaultBranch}</span>
                </span>
              }
              searchPlaceholder="Search branches..."
              emptyMessage={isSearching ? "Searching..." : "No branches found."}
              creatable
              leftIcon={
                isSearching ? (
                  <div className="animate-spin h-3 w-3 border border-gray-6 border-t-gray-11 rounded-full" />
                ) : (
                  <Magnifier className="text-gray-9 size-3" iconSize="sm-regular" />
                )
              }
              className="min-h-7! h-7! rounded-lg border-grayA-4 text-[13px] bg-transparent font-medium shadow-md"
              wrapperClassName="w-full"
              popoverClassName="w-[400px]"
            />
          )}
        </div>
        <Button
          variant="outline"
          className="rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all px-3"
          disabled={disabled || isLoading}
          loading={loading}
          onClick={() => onSelect(repo, selectedBranch)}
        >
          <span className="text-[13px] text-gray-12 font-medium">Select</span>
        </Button>
      </div>
    </div>
  );
};
