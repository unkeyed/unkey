import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useMemo } from "react";
import { ComboboxSkeleton, GitHubSettingCard, ManageGitHubAppLink, RepoNameLabel } from "./shared";

export const GitHubConnected = ({
  projectId,
  appId,
  onInstall,
  repoFullName,
}: {
  projectId: string;
  appId: string;
  onInstall: () => Promise<void> | void;
  repoFullName: string;
}) => {
  const utils = trpc.useUtils();

  const { data: reposData, isLoading: isLoadingRepos } = trpc.github.listRepositories.useQuery(
    { projectId },
    { refetchOnWindowFocus: false },
  );

  const repoOptions = useMemo(
    () =>
      (reposData?.repositories ?? []).map((repo) => ({
        value: `${repo.installationId}:${repo.id}`,
        label: <RepoNameLabel fullName={repo.fullName} />,
        searchValue: repo.fullName,
        selectedLabel: <RepoNameLabel fullName={repo.fullName} />,
      })),
    [reposData?.repositories],
  );

  const selectedValue = useMemo(() => {
    const match = reposData?.repositories.find((r) => r.fullName === repoFullName);
    return match ? `${match.installationId}:${match.id}` : "";
  }, [reposData?.repositories, repoFullName]);

  const selectRepoMutation = trpc.github.selectRepository.useMutation({
    onSuccess: async () => {
      toast.success("Repository connected");
      await utils.github.getInstallations.invalidate();
      await utils.github.getRepoTree.invalidate();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const handleSelectRepository = (value: string) => {
    const repo = reposData?.repositories.find((r) => `${r.installationId}:${r.id}` === value);
    if (!repo) {
      return;
    }
    selectRepoMutation.mutate({
      projectId,
      appId,
      repositoryId: repo.id,
      repositoryFullName: repo.fullName,
      installationId: repo.installationId,
    });
  };

  const expandable = (
    <div className="px-6 py-4 flex flex-col gap-3 bg-grayA-2 rounded-b-xl">
      <span className="text-gray-9 text-[13px]">
        Pushes to this repository will trigger deployments.
      </span>
      <div className="flex items-center gap-5 pt-1">
        <ManageGitHubAppLink
          onInstall={onInstall}
          variant="primary"
          text={<span>Manage GitHub</span>}
        />
      </div>
    </div>
  );

  return (
    <GitHubSettingCard expandable={expandable} chevronState="interactive">
      <div onClick={(e) => e.stopPropagation()} onKeyDown={(e) => e.stopPropagation()}>
        {isLoadingRepos ? (
          <ComboboxSkeleton />
        ) : (
          <Combobox
            className="w-[200px] text-left h-7 border-grayA-4"
            options={repoOptions}
            value={selectedValue}
            onSelect={handleSelectRepository}
            placeholder={<span className="text-left w-full">Select a repository...</span>}
            searchPlaceholder="Filter repositories..."
            disabled={selectRepoMutation.isLoading}
          />
        )}
      </div>
    </GitHubSettingCard>
  );
};
