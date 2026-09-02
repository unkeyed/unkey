"use client";

import { Github } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, SettingCard, Skeleton, toast } from "@unkey/ui";
import { useCallback } from "react";
import { ManageGitHubAppLink } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/components/build-settings/github-settings/shared";
import { trpc } from "@/lib/trpc/client";

type GithubConnectionState =
  | { status: "loading" }
  | { status: "connected" }
  | { status: "disconnected" };

export function GithubConnection() {
  const { data, isLoading } = trpc.github.hasInstallations.useQuery();
  const prepareInstall = trpc.github.prepareWorkspaceInstall.useMutation();

  const onInstall = useCallback(async () => {
    try {
      const { state } = await prepareInstall.mutateAsync();
      window.location.href = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to start GitHub install");
    }
  }, [prepareInstall]);

  const state: GithubConnectionState = isLoading
    ? { status: "loading" }
    : data?.hasInstallation
      ? { status: "connected" }
      : { status: "disconnected" };

  const description =
    state.status === "connected"
      ? "The Unkey GitHub App is installed on this workspace. Manage it on GitHub."
      : "Install the Unkey GitHub App on your workspace to deploy from your repositories.";

  const statusBadge =
    state.status === "connected" ? (
      <Badge variant="success">Connected</Badge>
    ) : state.status === "disconnected" ? (
      <Badge variant="secondary">Not connected</Badge>
    ) : null;

  return (
    <SettingCard
      title={
        <span className="flex items-center gap-2">
          GitHub
          {statusBadge}
        </span>
      }
      description={description}
      border="both"
      contentWidth="w-full lg:w-[420px] justify-end"
      icon={<Github className="size-4" />}
    >
      <div className="flex w-full items-center justify-end">
        {match(state)
          .with({ status: "loading" }, () => <Skeleton className="h-9 w-40 rounded-lg" />)
          .with({ status: "connected" }, () => (
            <ManageGitHubAppLink text="Manage on GitHub" onInstall={onInstall} />
          ))
          .with({ status: "disconnected" }, () => (
            <ManageGitHubAppLink text="Install GitHub App" onInstall={onInstall} />
          ))
          .exhaustive()}
      </div>
    </SettingCard>
  );
}
