"use client";

import { Github } from "@unkey/icons";
import { Button, SettingCard } from "@unkey/ui";
import { useParams } from "next/navigation";

type Props = {
  projectId: string;
  hasInstallations: boolean;
};

export const GitHubAppCard: React.FC<Props> = ({ projectId, hasInstallations }) => {
  const params = useParams<{ workspaceSlug: string }>();

  const handleConnectGitHub = () => {
    const state = `${projectId}:${params?.workspaceSlug ?? ""}`;
    const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
    window.location.href = installUrl;
  };

  return (
    <SettingCard
      title="GitHub App"
      description={
        hasInstallations
          ? "The Unkey GitHub App is installed. You can add more GitHub organizations or manage existing installations."
          : "Install the Unkey GitHub App to enable automatic deployments on push."
      }
      border={hasInstallations ? "top" : "both"}
      contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
    >
      <div className="flex justify-end gap-2">
        <Button variant={hasInstallations ? "outline" : "primary"} onClick={handleConnectGitHub}>
          <Github className="size-4" />
          {hasInstallations ? "Configure" : "Install"}
        </Button>
      </div>
    </SettingCard>
  );
};
