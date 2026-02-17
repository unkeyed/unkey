"use client";

import { trpc } from "@/lib/trpc/client";
import { useProjectData } from "../../../../data-provider";
import { GitHubConnected } from "./github-connected";
import { GitHubNoRepo } from "./github-no-repo";
import { ComboboxSkeleton, GitHubSettingCard, ManageGitHubAppLink } from "./shared";

type GitHubConnectionState =
  | { status: "loading" }
  | { status: "no-app"; installUrl: string }
  | { status: "no-repo"; installUrl: string }
  | { status: "connected"; repoFullName: string; repositoryId: number; installUrl: string };

export const GitHubSettings = () => {
  const { projectId } = useProjectData();

  const state = JSON.stringify({ projectId });
  const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;

  const { data, isLoading } = trpc.github.getInstallations.useQuery(
    { projectId },
    { staleTime: 0, refetchOnWindowFocus: true },
  );

  const connectionState: GitHubConnectionState = (() => {
    if (isLoading) {
      return { status: "loading" };
    }
    const hasInstallations = (data?.installations?.length ?? 0) > 0;
    if (!hasInstallations) {
      return { status: "no-app", installUrl };
    }
    const repoFullName = data?.repoConnection?.repositoryFullName;
    if (repoFullName) {
      const repositoryId = data?.repoConnection?.repositoryId ?? 0;
      return { status: "connected", repoFullName, repositoryId, installUrl };
    }
    return { status: "no-repo", installUrl };
  })();

  switch (connectionState.status) {
    case "loading":
      return (
        <GitHubSettingCard chevronState="disabled">
          <ComboboxSkeleton />
        </GitHubSettingCard>
      );
    // No-app means user haven't connected an app to unkey yet
    case "no-app":
      return (
        <GitHubSettingCard chevronState="interactive">
          <ManageGitHubAppLink
            installUrl={connectionState.installUrl}
            variant="outline"
            className="px-2.5 py-3 text-gray-12 font-medium text-[13px] bg-grayA-2 shadow-md hover:bg-grayA-3"
          />
        </GitHubSettingCard>
      );
    // User connected to unkey, but haven't selected a repo yet
    case "no-repo":
      return <GitHubNoRepo projectId={projectId} installUrl={connectionState.installUrl} />;
    case "connected":
      return (
        <GitHubConnected
          projectId={projectId}
          installUrl={connectionState.installUrl}
          repoFullName={connectionState.repoFullName}
          repositoryId={connectionState.repositoryId}
        />
      );
  }
};
