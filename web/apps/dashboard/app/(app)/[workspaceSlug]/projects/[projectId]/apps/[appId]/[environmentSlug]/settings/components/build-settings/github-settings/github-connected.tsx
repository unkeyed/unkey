import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Combobox, toast } from "@unkey/ui";
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
      await collection.apps.utils.refetch();
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

  return (
    <GitHubSettingCard>
      <div className="flex flex-col items-start gap-2">
        {isLoadingRepos ? (
          <ComboboxSkeleton />
        ) : (
          <Combobox
            className="w-[200px] text-left h-7"
            options={repoOptions}
            value={selectedValue}
            onSelect={handleSelectRepository}
            placeholder={<span className="text-left w-full">Select a repository...</span>}
            searchPlaceholder="Filter repositories..."
            disabled={selectRepoMutation.isLoading}
          />
        )}
        <span className="text-gray-9 text-sm">
          Pushes to this repository will trigger deployments.
        </span>
        <div className="flex items-center">
          <ManageGitHubAppLink
            onInstall={onInstall}
            variant="primary"
            text={<span>Manage GitHub</span>}
          />
        </div>
      </div>
    </GitHubSettingCard>
  );
};
