"use client";

import { Github } from "@unkey/icons";
import { SettingCard, buttonVariants } from "@unkey/ui";
import { useProjectData } from "../../data-provider";

type Props = {
  hasInstallations: boolean;
};

export const GitHubAppCard: React.FC<Props> = ({ hasInstallations }) => {
  const { projectId } = useProjectData();
  const state = JSON.stringify({ projectId });
  const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;

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
        <a
          href={installUrl}
          className={buttonVariants({
            variant: hasInstallations ? "outline-solid" : "primary",
          })}
        >
          <Github className="size-4" />
          {hasInstallations ? "Configure" : "Install"}
        </a>
      </div>
    </SettingCard>
  );
};
