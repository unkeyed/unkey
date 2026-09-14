import type { Deployment } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import { CodeBranch, CodeCommit, Layers2 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge } from "@unkey/ui";
import type { ComponentProps, ReactNode } from "react";

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
      <SourceDescription>
        <Layers2 iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span
          className="truncate"
          title={deployment.requestedImage ?? deployment.resolvedImage ?? undefined}
        >
          {deployment.requestedImage ?? deployment.resolvedImage ?? "Container image deployment"}
        </span>
      </SourceDescription>
    ))
    .with("git", () => (
      <SourceDescription>
        <CodeCommit iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span className="truncate">
          {deployment.gitCommitMessage || `${isCurrent ? "Current active" : "Target"} deployment`}
        </span>
      </SourceDescription>
    ))
    .with("unknown", () => (
      <SourceDescription>
        <Layers2 iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span>{isCurrent ? "Current active deployment" : "Target deployment"}</span>
      </SourceDescription>
    ))
    .exhaustive();
};

const SourceDescription = ({ children }: { children: ReactNode }) => (
  <div className="text-xs text-grayA-9 flex items-center gap-1.5 min-w-0">{children}</div>
);

const MetadataPill = ({ className, ...props }: ComponentProps<"div">) => (
  <div
    className={cn(
      "flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11",
      className,
    )}
    {...props}
  />
);

const DeploymentMetadata = ({ deployment }: Pick<DeploymentCardProps, "deployment">) => {
  return match(deployment.source)
    .with("oci", () => {
      if (!deployment.resolvedImage) {
        return null;
      }

      const digest = deployment.resolvedImage.split("@").at(-1) ?? deployment.resolvedImage;
      const digestLabel = digest.startsWith("sha256:") ? `sha256:${digest.slice(7, 19)}` : digest;
      return (
        <MetadataPill className="max-w-[180px]" title={deployment.resolvedImage}>
          <Layers2 iconSize="sm-regular" className="shrink-0 text-gray-12" />
          <span className="truncate font-mono">{digestLabel}</span>
        </MetadataPill>
      );
    })
    .with("git", () => (
      <div className="flex gap-1.5">
        {deployment.gitBranch && (
          <MetadataPill className="max-w-[100px]">
            <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-12" />
            <span className="truncate">{deployment.gitBranch}</span>
          </MetadataPill>
        )}
        {deployment.gitCommitSha && (
          <MetadataPill>
            <CodeCommit iconSize="sm-regular" className="shrink-0 text-gray-12" />
            <span>{shortenId(deployment.gitCommitSha)}</span>
          </MetadataPill>
        )}
      </div>
    ))
    .with("unknown", () => null)
    .exhaustive();
};
