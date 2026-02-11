"use client";

import { RepoDisplay } from "@/app/(app)/[workspaceSlug]/projects/_components/list/repo-display";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingCard,
  toast,
} from "@unkey/ui";
import { useProjectData } from "../../data-provider";

type Props = {
  connectedRepo: string | null;
  onDisconnect: () => void;
  isDisconnecting: boolean;
};

export const RepositoryCard: React.FC<Props> = ({
  connectedRepo,
  onDisconnect,
  isDisconnecting,
}) => {
  const { projectId } = useProjectData();
  const utils = trpc.useUtils();

  const { data: reposData, isLoading: isLoadingRepos } = trpc.github.listRepositories.useQuery(
    { projectId },
    {
      enabled: !connectedRepo,
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

  const handleSelectRepository = (value: string) => {
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

  if (connectedRepo) {
    return (
      <SettingCard
        title="Repository"
        description={
          <div className="flex flex-col gap-1">
            <RepoDisplay
              url={`https://github.com/${connectedRepo}`}
              className="text-accent-11 text-[13px]"
            />
            <span className="text-gray-9 text-[13px]">
              Pushes to this repository will trigger deployments.
            </span>
          </div>
        }
        border="bottom"
        contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
      >
        <div className="flex justify-end">
          <Button variant="outline" color="danger" onClick={onDisconnect} loading={isDisconnecting}>
            Disconnect
          </Button>
        </div>
      </SettingCard>
    );
  }

  return (
    <SettingCard
      title="Repository"
      description={
        <div className="flex flex-col gap-1">
          <span>Select a repository to connect to this project.</span>
          <span className="text-gray-9 text-[13px]">
            Pushes to this repository will trigger deployments.
          </span>
        </div>
      }
      border="bottom"
      contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
    >
      <div className="flex justify-end w-full max-w-[280px]">
        {isLoadingRepos ? (
          <div className="h-9 w-full bg-grayA-3 animate-pulse rounded-lg" />
        ) : reposData?.repositories.length ? (
          <Select onValueChange={handleSelectRepository} disabled={selectRepoMutation.isLoading}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Select a repository..." />
            </SelectTrigger>
            <SelectContent>
              {reposData.repositories.map((repo) => (
                <SelectItem
                  key={`${repo.installationId}:${repo.id}`}
                  value={`${repo.installationId}:${repo.id}`}
                >
                  {repo.fullName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <span className="text-gray-9 text-sm">No repositories found.</span>
        )}
      </div>
    </SettingCard>
  );
};
