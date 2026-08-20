import type { Deployment } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { CodeBranch, CodeCommit, Cube } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge } from "@unkey/ui";

type DeploymentCardProps = {
  deployment: Deployment;
  isCurrent: boolean;
};

export const DeploymentCard = ({ deployment, isCurrent }: DeploymentCardProps) => {
  return (
    <div className="bg-white dark:bg-black border border-grayA-4 rounded-[14px] p-4 relative">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs text-accent-12 font-semibold font-mono">{deployment.id}</span>
            <Badge
              variant={isCurrent ? "success" : "primary"}
              className={`px-1.5 capitalize ${isCurrent ? "text-successA-11" : "text-grayA-11"}`}
            >
              {isCurrent ? "Current" : deployment.status}
            </Badge>
          </div>
          <DeploymentDescription deployment={deployment} isCurrent={isCurrent} />
        </div>
        <DeploymentMetadata deployment={deployment} />
      </div>
    </div>
  );
};

const DeploymentDescription = ({
  deployment,
  isCurrent,
}: Pick<DeploymentCardProps, "deployment" | "isCurrent">) => {
  return match(deployment.source)
    .with("oci", () => (
      <div className="text-xs text-grayA-9 flex items-center gap-1.5 min-w-0">
        <Cube iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span
          className="truncate"
          title={deployment.requestedImage ?? deployment.resolvedImage ?? undefined}
        >
          {deployment.requestedImage ?? deployment.resolvedImage ?? "OCI image deployment"}
        </span>
      </div>
    ))
    .with("git", () => (
      <div className="text-xs text-grayA-9 flex items-center gap-1.5 min-w-0">
        <CodeCommit iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span className="truncate">
          {deployment.gitCommitMessage || `${isCurrent ? "Current active" : "Target"} deployment`}
        </span>
      </div>
    ))
    .with("unknown", () => (
      <div className="text-xs text-grayA-9 flex items-center gap-1.5">
        <Cube iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span>{isCurrent ? "Current active deployment" : "Target deployment"}</span>
      </div>
    ))
    .exhaustive();
};

const DeploymentMetadata = ({ deployment }: Pick<DeploymentCardProps, "deployment">) => {
  return match(deployment.source)
    .with("oci", () => {
      if (!deployment.resolvedImage) {
        return null;
      }

      const digest = deployment.resolvedImage.split("@").at(-1) ?? deployment.resolvedImage;
      const digestLabel = digest.startsWith("sha256:") ? `sha256:${digest.slice(7, 19)}` : digest;
      return (
        <div
          className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11 max-w-[180px]"
          title={deployment.resolvedImage}
        >
          <Cube iconSize="sm-regular" className="shrink-0 text-gray-12" />
          <span className="truncate font-mono">{digestLabel}</span>
        </div>
      );
    })
    .with("git", () => (
      <div className="flex gap-1.5">
        {deployment.gitBranch && (
          <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11 max-w-[100px]">
            <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-12" />
            <span className="truncate">{deployment.gitBranch}</span>
          </div>
        )}
        {deployment.gitCommitSha && (
          <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11">
            <CodeCommit iconSize="sm-regular" className="shrink-0 text-gray-12" />
            <span>{shortenId(deployment.gitCommitSha)}</span>
          </div>
        )}
      </div>
    ))
    .with("unknown", () => null)
    .exhaustive();
};
