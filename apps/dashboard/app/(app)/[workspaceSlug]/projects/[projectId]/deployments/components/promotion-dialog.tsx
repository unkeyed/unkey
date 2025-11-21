"use client";

import { type Deployment, collection, collectionManager } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { CircleInfo, CodeBranch, CodeCommit, Link4 } from "@unkey/icons";
import { Badge, Button, DialogContainer, toast } from "@unkey/ui";
import { StatusIndicator } from "../../details/active-deployment-card/status-indicator";

type DeploymentSectionProps = {
  title: string;
  deployment: Deployment;
  isLive: boolean;
  showSignal?: boolean;
};

const DeploymentSection = ({ title, deployment, isLive, showSignal }: DeploymentSectionProps) => (
  <div className="space-y-2">
    <div className="flex items-center gap-2">
      <h3 className="text-[13px] text-grayA-11">{title}</h3>
      <CircleInfo iconSize="sm-regular" className="text-gray-9" />
    </div>
    <DeploymentCard deployment={deployment} isLive={isLive} showSignal={showSignal} />
  </div>
);

type PromotionDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  targetDeployment: Deployment;
  liveDeployment: Deployment;
};

export const PromotionDialog = ({
  isOpen,
  onClose,
  targetDeployment,
  liveDeployment,
}: PromotionDialogProps) => {
  const utils = trpc.useUtils();
  const domainCollection = collectionManager.getProjectCollections(
    liveDeployment.projectId,
  ).domains;
  const domains = useLiveQuery((q) =>
    q
      .from({ domain: domainCollection })
      .where(({ domain }) => inArray(domain.sticky, ["environment", "live"]))
      .where(({ domain }) => eq(domain.deploymentId, liveDeployment.id)),
  );
  const promote = trpc.deploy.deployment.promote.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Promotion completed", {
        description: `Successfully promoted to deployment ${targetDeployment.id}`,
      });
      // hack to revalidate
      try {
        // @ts-expect-error Their docs say it's here
        collection.projects.utils.refetch();
        // @ts-expect-error Their docs say it's here
        collection.deployments.utils.refetch();
        // @ts-expect-error Their docs say it's here
        collection.domains.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }

      onClose();
    },
    onError: (error) => {
      toast.error("Promotion failed", {
        description: error.message,
      });
    },
  });

  const handlePromotion = async () => {
    await promote
      .mutateAsync({
        targetDeploymentId: targetDeployment.id,
      })
      .catch((error) => {
        console.error("Promotion error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Promotion to version"
      subTitle="Switch the active deployment to a target stable version"
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handlePromotion}
          disabled={promote.isLoading}
          loading={promote.isLoading}
          className="w-full rounded-lg"
        >
          Promote to
          {targetDeployment.gitCommitSha
            ? shortenId(targetDeployment.gitCommitSha)
            : targetDeployment.id}
        </Button>
      }
    >
      <div className="space-y-9">
        <DeploymentSection
          title="Live Deployment"
          deployment={liveDeployment}
          isLive={true}
          showSignal={true}
        />
        <div>
          {domains.data.map((domain) => (
            <div className="space-y-2" key={domain.id}>
              <div className="flex items-center gap-2">
                <h3 className="text-[13px] text-grayA-11">Domain</h3>
                <CircleInfo iconSize="sm-regular" className="text-gray-9" />
              </div>
              <div className="bg-white dark:bg-black border border-grayA-5 rounded-lg p-4 relative">
                <div className="flex items-center">
                  <Link4 className="text-gray-9" iconSize="sm-medium" />
                  <div className="text-gray-12 font-medium text-xs ml-3 mr-2">
                    {domain.hostname}
                  </div>
                  <div className="ml-3" />
                </div>
              </div>
            </div>
          ))}
        </div>
        <DeploymentSection title="Target Deployment" deployment={targetDeployment} isLive={false} />
      </div>
    </DialogContainer>
  );
};

type DeploymentCardProps = {
  deployment: Deployment;
  isLive: boolean;
  showSignal?: boolean;
};

const DeploymentCard = ({ deployment, isLive, showSignal }: DeploymentCardProps) => (
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
