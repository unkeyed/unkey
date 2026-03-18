import { trpc } from "@/lib/trpc/client";
import { Check, ChevronDown, CodeBranch } from "@unkey/icons";
import {
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  TimestampInfo,
} from "@unkey/ui";
import { useState } from "react";
import { LanguageIcon } from "./language-icon";

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
  onSelect: (repo: RepoItem) => void;
  disabled: boolean;
  loading: boolean;
}) => {
  const [owner, repoName] = repo.fullName.split("/");
  const [selectedBranch, setSelectedBranch] = useState(repo.defaultBranch);

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
      <div className="flex gap-2 items-center ml-auto">
        {isLoading ? (
          <div className="h-4 w-28 bg-grayA-3 rounded animate-pulse" />
        ) : details.hasDockerfile ? (
          <>
            <Check className="text-success-9" iconSize="sm-regular" />
            <span className="text-gray-10 text-xs">Dockerfile detected</span>
          </>
        ) : (
          <span className="text-gray-10 text-xs">No Dockerfile</span>
        )}
      </div>
      <div className="flex gap-2 items-center">
        <div className="ml-6 w-35">
          {isLoading ? (
            <div className="h-8 w-full bg-grayA-3 rounded-lg animate-pulse" />
          ) : (
            <Select value={selectedBranch} onValueChange={setSelectedBranch}>
              <SelectTrigger
                className="min-h-7! h-7! rounded-lg border-grayA-4 text-[13px] bg-transparent w-full font-medium shadow-md"
                wrapperClassName="w-full"
                leftIcon={<CodeBranch className="text-gray-12 size-3" iconSize="sm-regular" />}
                rightIcon={<ChevronDown className="text-gray-9 size-3 right-2 absolute" />}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="max-h-50">
                {(details.branches ?? []).map((branch) => (
                  <SelectItem
                    key={branch.name}
                    value={branch.name}
                    className="cursor-pointer text-[13px]"
                  >
                    {branch.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>
        <Button
          variant="outline"
          className="rounded-lg border-grayA-4 shadow-md transition-all h-7"
          disabled={disabled || isLoading}
          loading={loading}
          onClick={() => onSelect(repo)}
        >
          <span className="text-[13px] text-gray-12 font-medium">Select</span>
        </Button>
      </div>
    </div>
  );
};
