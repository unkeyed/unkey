"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import { useProject } from "../../layout-provider";
import { ActiveDeploymentCardEmpty } from "./components/active-deployment-card-empty";
import { Avatar } from "../../components/git-avatar";
import { InfoChip } from "../../components/info-chip";
import { ActiveDeploymentCardSkeleton } from "./components/skeleton";
import { StatusIndicator } from "../../components/status-indicator";
import { Card } from "../card";

type Props = {
  deploymentId: string | null;
  statusBadge?: React.ReactNode;
  trailingContent?: React.ReactNode;
  expandableContent?: React.ReactNode;
};

export const ActiveDeploymentCard = ({
  deploymentId,
  statusBadge,
  trailingContent,
  expandableContent,
}: Props) => {
  const { collections } = useProject();
  const { data, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [deploymentId],
  );
  const deployment = data.at(0);

  if (isLoading) {
    return <ActiveDeploymentCardSkeleton />;
  }
  if (!deployment) {
    return <ActiveDeploymentCardEmpty />;
  }

  return (
    <Card className="rounded-[14px] pt-[14px] flex justify-between flex-col overflow-hidden border-gray-4">
      <div className="flex w-full justify-between items-center px-[22px]">
        <div className="flex gap-5 items-center">
          <StatusIndicator withSignal />
          <div className="flex flex-col gap-1">
            <div className="text-accent-12 font-medium text-xs">{deployment.id}</div>
            <div className="text-gray-9 text-xs">{deployment.gitCommitMessage}</div>
          </div>
        </div>
        <div className="flex items-center gap-4">
          {statusBadge}
          <div className="items-center flex gap-2">
            <div className="flex gap-2 items-center">
              <span className="text-gray-9 text-xs">Created by</span>
              <Avatar src={deployment.gitCommitAuthorAvatarUrl} alt="Author" />
              <span className="font-medium text-grayA-12 text-xs">
                {deployment.gitCommitAuthorHandle}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-gray-1 rounded-b-[14px]">
        <div className="relative h-4 flex items-center justify-center">
          <div className="absolute top-0 left-0 right-0 h-4 border-b border-gray-4 rounded-b-[14px] bg-white dark:bg-black" />
        </div>

        <div className="pb-2.5 pt-2 flex justify-between items-center px-3">
          <div className="flex items-center gap-2.5">
            <TimestampInfo
              value={deployment.createdAt}
              displayType="relative"
              className="text-grayA-9 text-xs"
            />
            <div className="flex items-center gap-1.5">
              <InfoChip icon={CodeBranch}>
                <span className="text-grayA-9 text-xs truncate max-w-32">
                  {deployment.gitBranch}
                </span>
              </InfoChip>
              <InfoChip icon={CodeCommit}>
                <span className="text-grayA-9 text-xs">
                  {(deployment.gitCommitSha ?? "").slice(0, 7)}
                </span>
              </InfoChip>
            </div>
          </div>
          {trailingContent}
        </div>

        {expandableContent}
      </div>
    </Card>
  );
};
