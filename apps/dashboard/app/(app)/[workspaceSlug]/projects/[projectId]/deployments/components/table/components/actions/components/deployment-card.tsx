import { StatusIndicator } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/details/active-deployment-card/status-indicator";
import type { Deployment } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { Badge } from "@unkey/ui";

type DeploymentCardProps = {
  deployment: Deployment;
  isLive: boolean;
  showSignal?: boolean;
};

export const DeploymentCard = ({ deployment, isLive, showSignal }: DeploymentCardProps) => (
  <div className="bg-white dark:bg-black border border-grayA-5 rounded-lg p-4 relative">
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <StatusIndicator withSignal={showSignal} />
        <div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-accent-12 font-mono">
              {`${deployment.id.slice(0, 3)}...${deployment.id.slice(-4)}`}
            </span>
            <Badge
              variant={isLive ? "success" : "primary"}
              className={`px-1.5 capitalize ${isLive ? "text-successA-11" : "text-grayA-11"}`}
            >
              {isLive ? "Live" : deployment.status}
            </Badge>
          </div>
          <div className="text-xs text-grayA-9">
            {deployment.gitCommitMessage || `${isLive ? "Current active" : "Target"} deployment`}
          </div>
        </div>
      </div>
      <div className="flex gap-1.5">
        <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11 max-w-[100px]">
          <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-12" />
          <span className="truncate">{deployment.gitBranch}</span>
        </div>
        <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11">
          <CodeCommit iconSize="sm-regular" className="shrink-0 text-gray-12" />
          <span>{shortenId(deployment.gitCommitSha ?? "")}</span>
        </div>
      </div>
    </div>
  </div>
);
