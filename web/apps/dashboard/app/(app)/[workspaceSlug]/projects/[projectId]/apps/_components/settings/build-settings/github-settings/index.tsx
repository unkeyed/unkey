"use client";

import { githubUrl } from "@/lib/github-url";
import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import { toast } from "@unkey/ui";
import { useCallback } from "react";
import { useBuildSource } from "../../hooks/use-build-source";
import { GitHubConnected } from "./github-connected";
import { GitHubNoRepo } from "./github-no-repo";
import { ComboboxSkeleton, GitHubSettingCard, ManageGitHubAppLink, RepoNameLabel } from "./shared";

type GitHubConnectionState =
  | { status: "loading" }
  | { status: "no-app"; onInstall: () => Promise<void> }
  | { status: "no-repo"; appId: string; onInstall: () => Promise<void> }
  | {
      status: "connected";
      appId: string;
      repoFullName: string;
      repositoryId: number;
      onInstall: () => Promise<void>;
    };

type GitHubProps = {
  readOnly?: boolean;
};

export const GitHub = ({ readOnly = false }: GitHubProps) => {
  const {
    projectId,
    appId,
    installations: { data, isLoading },
  } = useBuildSource({ staleTime: 0, refetchOnWindowFocus: true });

  // The state on the GitHub install URL is a server-signed token bound to
  // this user, workspace, and project. Computing it requires a tRPC round
  // trip, so we mint it lazily on click rather than in render.
  const prepareInstallation = trpc.github.prepareInstallation.useMutation();
  const onInstall = useCallback(async () => {
    try {
      const { state } = await prepareInstallation.mutateAsync({
        projectId,
        appId,
        returnTo: "settings",
      });
      window.location.href = githubUrl.install(state);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to start GitHub install");
    }
  }, [projectId, appId, prepareInstallation]);

  const connectionState: GitHubConnectionState = (() => {
    if (isLoading) {
      return { status: "loading" };
    }
    const hasInstallations = (data?.installations?.length ?? 0) > 0;
    if (!hasInstallations) {
      return { status: "no-app", onInstall };
    }
    const appId = data?.appId;
    if (!appId) {
      return { status: "no-app", onInstall };
    }
    const repoFullName = data?.repoConnection?.repositoryFullName;
    if (repoFullName) {
      const repositoryId = data?.repoConnection?.repositoryId ?? 0;
      return { status: "connected", appId, repoFullName, repositoryId, onInstall };
    }
    return { status: "no-repo", appId, onInstall };
  })();

  return match(connectionState)
    .with({ status: "loading" }, () => (
      <GitHubSettingCard>
        <ComboboxSkeleton />
      </GitHubSettingCard>
    ))
    .with({ status: "no-app" }, ({ onInstall: install }) => (
      <GitHubSettingCard>
        <div className="flex items-center">
          <ManageGitHubAppLink
            onInstall={install}
            variant="outline"
            className="px-2.5 py-3 text-gray-12 font-medium text-sm hover:bg-grayA-2"
            text="Connect GitHub"
          />
        </div>
      </GitHubSettingCard>
    ))
    .with({ status: "no-repo" }, ({ appId, onInstall: install }) => (
      <GitHubNoRepo projectId={projectId} appId={appId} onInstall={install} />
    ))
    .with({ status: "connected" }, ({ appId, repoFullName, onInstall: install }) => {
      if (readOnly) {
        return (
          <GitHubSettingCard>
            <RepoNameLabel fullName={repoFullName} />
          </GitHubSettingCard>
        );
      }
      return (
        <GitHubConnected
          projectId={projectId}
          appId={appId}
          onInstall={install}
          repoFullName={repoFullName}
        />
      );
    })
    .exhaustive();
};
