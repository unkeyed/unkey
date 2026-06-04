"use client";

import { trpc } from "@/lib/trpc/client";
import { SettingsZoneRow, toast } from "@unkey/ui";
import { useAppId, useProjectData } from "../../data-provider";

export function DisconnectGitHub() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const utils = trpc.useUtils();

  const { data } = trpc.github.getInstallations.useQuery({ projectId, appId }, { staleTime: 0 });

  const isConnected = Boolean(data?.repoConnection?.repositoryFullName);

  const disconnectRepoMutation = trpc.github.disconnectRepo.useMutation({
    onSuccess: async () => {
      toast.success("Repository disconnected");
      await utils.github.getInstallations.invalidate();
      await utils.github.getRepoTree.invalidate();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  if (!isConnected) {
    return null;
  }

  return (
    <SettingsZoneRow
      title="Disconnect repository"
      description="Deployments will no longer be triggered by pushes to this repository."
      action={{
        label: "Disconnect repository",
        onClick: () => disconnectRepoMutation.mutate({ appId }),
        loading: disconnectRepoMutation.isLoading,
      }}
    />
  );
}
