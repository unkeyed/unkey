"use client";

import { trpc } from "@/lib/trpc/client";
import { Loading, toast } from "@unkey/ui";
import { useProjectData } from "../../data-provider";
import { GitHubAppCard } from "./github-app-card";
import { RepositoryCard } from "./repository-card";

export const GitHubSettingsClient: React.FC = () => {
  const { projectId } = useProjectData();
  const utils = trpc.useUtils();

  const { data, isLoading, refetch } = trpc.github.getInstallations.useQuery(
    { projectId },
    {
      staleTime: 0,
      refetchOnWindowFocus: true,
    },
  );

  const disconnectRepoMutation = trpc.github.disconnectRepo.useMutation({
    onSuccess: async () => {
      toast.success("Repository disconnected");
      await utils.github.getInstallations.invalidate();
      await refetch();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  if (isLoading) {
    return (
      <div className="w-full flex items-center justify-center py-12">
        <Loading />
      </div>
    );
  }

  const hasInstallations = (data?.installations?.length ?? 0) > 0;
  const repoConnection = data?.repoConnection;

  return (
    <div>
      {hasInstallations ? (
        <>
          <GitHubAppCard hasInstallations={true} />
          <RepositoryCard
            connectedRepo={repoConnection?.repositoryFullName ?? null}
            onDisconnect={() => disconnectRepoMutation.mutate({ projectId })}
            isDisconnecting={disconnectRepoMutation.isLoading}
          />
        </>
      ) : (
        <GitHubAppCard hasInstallations={false} />
      )}
    </div>
  );
};
