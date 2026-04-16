"use client";

import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import { useProjectData } from "../../../../data-provider";
import { SelectedConfig } from "../../shared/selected-config";
import { GitHubConnected } from "./github-connected";
import { GitHubNoRepo } from "./github-no-repo";
import { ComboboxSkeleton, GitHubSettingCard, ManageGitHubAppLink, RepoNameLabel } from "./shared";

type GitHubConnectionState =
  | { status: "loading" }
  | { status: "no-app"; installUrl: string }
  | { status: "no-repo"; appId: string; installUrl: string }
  | {
      status: "connected";
      appId: string;
      repoFullName: string;
      repositoryId: number;
      installUrl: string;
    };

type GitHubProps = {
  readOnly?: boolean;
  onBeforeNavigate?: () => void;
};

export const GitHub = ({ readOnly = false, onBeforeNavigate }: GitHubProps) => {
  const { projectId } = useProjectData();

  const state = JSON.stringify({ projectId, returnTo: "settings" });
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
    const appId = data?.appId;
    if (!appId) {
      return { status: "no-app", installUrl };
    }
    const repoFullName = data?.repoConnection?.repositoryFullName;
    if (repoFullName) {
      const repositoryId = data?.repoConnection?.repositoryId ?? 0;
      return { status: "connected", appId, repoFullName, repositoryId, installUrl };
    }
    return { status: "no-repo", appId, installUrl };
  })();

  return match(connectionState)
    .with({ status: "loading" }, () => (
      <GitHubSettingCard chevronState="disabled">
        <ComboboxSkeleton />
      </GitHubSettingCard>
    ))
    .with({ status: "no-app" }, ({ installUrl: url }) => (
      <GitHubSettingCard chevronState="disabled">
        <ManageGitHubAppLink
          installUrl={url}
          variant="outline"
          className="px-2.5 py-3 text-gray-12 font-medium text-[13px] hover:bg-grayA-2"
          onBeforeNavigate={onBeforeNavigate}
        />
      </GitHubSettingCard>
    ))
    .with({ status: "no-repo" }, ({ appId, installUrl: url }) => (
      <GitHubNoRepo
        projectId={projectId}
        appId={appId}
        installUrl={url}
        onBeforeNavigate={onBeforeNavigate}
      />
    ))
    .with({ status: "connected" }, ({ appId, repoFullName, installUrl: url }) => {
      if (readOnly) {
        return (
          <GitHubSettingCard chevronState="disabled">
            <SelectedConfig label={<RepoNameLabel fullName={repoFullName} />} />
          </GitHubSettingCard>
        );
      }
      return (
        <GitHubConnected
          projectId={projectId}
          appId={appId}
          installUrl={url}
          repoFullName={repoFullName}
          onBeforeNavigate={onBeforeNavigate}
        />
      );
    })
    .exhaustive();
};
