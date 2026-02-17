import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useMemo, useState } from "react";
import { ComboboxSkeleton, GitHubSettingCard, ManageGitHubAppLink, RepoNameLabel } from "./shared";

export const GitHubNoRepo = ({
  projectId,
  installUrl,
}: {
  projectId: string;
  installUrl: string;
}) => {
  const utils = trpc.useUtils();
  const [selectedRepo, setSelectedRepo] = useState("");

  const { data: reposData, isLoading: isLoadingRepos } = trpc.github.listRepositories.useQuery(
    {
      projectId,
    },
    {
      refetchOnWindowFocus: false,
    },
  );

  const selectRepoMutation = trpc.github.selectRepository.useMutation({
    onSuccess: async () => {
      toast.success("Repository connected");
      await utils.github.getInstallations.invalidate();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const repoOptions = useMemo(
    () =>
      (reposData?.repositories ?? []).map((repo) => ({
        value: `${repo.installationId}:${repo.id}`,
        label: repo.fullName,
        searchValue: repo.fullName,
        selectedLabel: <RepoNameLabel fullName={repo.fullName} />,
      })),
    [reposData?.repositories],
  );

  const handleSelectRepository = (value: string) => {
    setSelectedRepo(value);
    const repo = reposData?.repositories.find((r) => `${r.installationId}:${r.id}` === value);
    if (!repo) {
      return;
    }
    selectRepoMutation.mutate({
      projectId,
      repositoryId: repo.id,
      repositoryFullName: repo.fullName,
      installationId: repo.installationId,
    });
  };

  const collapsed = (
    <div onClick={(e) => e.stopPropagation()} onKeyDown={(e) => e.stopPropagation()}>
      {isLoadingRepos ? (
        <ComboboxSkeleton />
      ) : repoOptions.length ? (
        <Combobox
          className="w-[250px] text-left min-h-8"
          options={repoOptions}
          value={selectedRepo}
          onSelect={handleSelectRepository}
          placeholder=<span className="text-left w-full">Select a repository...</span>
          searchPlaceholder="Filter repositories..."
          disabled={selectRepoMutation.isLoading}
        />
      ) : (
        <ManageGitHubAppLink
          installUrl={installUrl}
          variant="outline"
          className="ml-0 h-8 px-3 py-2 rounded-lg"
          text={
            <>
              <span className="text-gray-9">Import from</span>
              <span className="text-gray-12 font-medium"> GitHub</span>
            </>
          }
        />
      )}
    </div>
  );

  return <GitHubSettingCard chevronState="disabled">{collapsed}</GitHubSettingCard>;
};
