"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { match } from "@unkey/match";
import { SettingsZoneRow, toast } from "@unkey/ui";
import { useAppId, useProjectData } from "../../data-provider";

export function DisconnectGitHub() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const utils = trpc.useUtils();
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appQuery.data?.[0];
  const shouldLoadGitHub = app
    ? match(app.sourceType)
        .with("github", () => true)
        .with("docker_image", () => false)
        .with("legacy", () => Boolean(app.repositoryFullName))
        .exhaustive()
    : false;

  const { data } = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: shouldLoadGitHub, staleTime: 0 },
  );

  const isConnected = shouldLoadGitHub && Boolean(data?.repoConnection?.repositoryFullName);

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
