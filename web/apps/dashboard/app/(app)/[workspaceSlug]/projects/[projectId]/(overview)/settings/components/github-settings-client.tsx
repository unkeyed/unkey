"use client";

import { trpc } from "@/lib/trpc/client";
import { Loading, toast } from "@unkey/ui";
import { GitHubAppCard } from "./github-app-card";
import { RepositoryCard } from "./repository-card";

type Props = {
  projectId: string;
};

export const GitHubSettingsClient: React.FC<Props> = ({ projectId }) => {
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
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
          <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
            Project Settings
          </div>
          <div className="w-full flex items-center justify-center py-12">
            <Loading />
          </div>
        </div>
      </div>
    );
  }

  const hasInstallations = (data?.installations?.length ?? 0) > 0;
  const repoConnection = data?.repoConnection;

  return (
    <div className="py-3 w-full flex items-center justify-center">
      <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
        <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          Project Settings
        </div>
        <div className="flex flex-col w-full gap-6">
          <div>
            {hasInstallations ? (
              <>
                <GitHubAppCard projectId={projectId} hasInstallations={true} />
                <RepositoryCard
                  projectId={projectId}
                  connectedRepo={repoConnection?.repositoryFullName ?? null}
                  onDisconnect={() => disconnectRepoMutation.mutate({ projectId })}
                  isDisconnecting={disconnectRepoMutation.isLoading}
                />
              </>
            ) : (
              <GitHubAppCard projectId={projectId} hasInstallations={false} />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
