"use client";

import { type Deployment, collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { inArray, useLiveQuery } from "@tanstack/react-db";
import { CircleInfo, CodeBranch, CodeCommit, Link4 } from "@unkey/icons";
import { Badge, Button, DialogContainer, toast } from "@unkey/ui";
import { StatusIndicator } from "../../details/active-deployment-card/status-indicator";
import { useProjectLayout } from "../../layout-provider";

type DeploymentSectionProps = {
  title: string;
  deployment: Deployment;
  isLive: boolean;
  showSignal?: boolean;
};

const DeploymentSection = ({
  title,
  deployment,
  isLive,
  showSignal,
}: DeploymentSectionProps) => (
  <div className="space-y-2">
    <div className="flex items-center gap-2">
      <h3 className="text-[13px] text-grayA-11">{title}</h3>
      <CircleInfo size="sm-regular" className="text-gray-9" />
    </div>
    <DeploymentCard
      deployment={deployment}
      isLive={isLive}
      showSignal={showSignal}
    />
  </div>
);

type RollbackDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  targetDeployment: Deployment;
  liveDeployment: Deployment;
};

export const RollbackDialog = ({
  isOpen,
  onClose,
  targetDeployment,
  liveDeployment,
}: RollbackDialogProps) => {
  const utils = trpc.useUtils();

  const {
    collections: { domains: domainCollection },
  } = useProjectLayout();
  const domains = useLiveQuery((q) =>
    q
      .from({ domain: domainCollection })
      .where(({ domain }) => inArray(domain.sticky, ["environment", "live"]))
  );

  const rollback = trpc.deploy.deployment.rollback.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Rollback completed", {
        description: `Successfully rolled back to deployment ${targetDeployment.id}`,
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
      toast.error("Rollback failed", {
        description: error.message,
      });
    },
  });

  const handleRollback = async () => {
    await rollback
      .mutateAsync({
        targetDeploymentId: targetDeployment.id,
      })
      .catch((error) => {
        console.error("Rollback error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Rollback to version"
      subTitle="Switch the active deployment to a target stable version"
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handleRollback}
          disabled={rollback.isLoading}
          loading={rollback.isLoading}
          className="w-full rounded-lg"
        >
          Rollback to target version
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
                <CircleInfo size="sm-regular" className="text-gray-9" />
              </div>
              <div className="bg-white dark:bg-black border border-grayA-5 rounded-lg p-4 relative">
                <div className="flex items-center">
                  <Link4 className="text-gray-9" size="sm-medium" />
                  <div className="text-gray-12 font-medium text-xs ml-3 mr-2">
                    {domain.domain}
                  </div>
                  <div className="ml-3" />
                </div>
              </div>
            </div>
          ))}
        </div>
        <DeploymentSection
          title="Target Deployment"
          deployment={targetDeployment}
          isLive={false}
        />
      </div>
    </DialogContainer>
  );
};

type DeploymentCardProps = {
  deployment: Deployment;
  isLive: boolean;
  showSignal?: boolean;
};

const DeploymentCard = ({
  deployment,
  isLive,
  showSignal,
}: DeploymentCardProps) => (
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
              className={`px-1.5 capitalize ${
                isLive ? "text-successA-11" : "text-grayA-11"
              }`}
            >
              {isLive ? "Live" : deployment.status}
            </Badge>
          </div>
          <div className="text-xs text-grayA-9">
            {deployment.gitCommitMessage ||
              `${isLive ? "Current active" : "Target"} deployment`}
          </div>
        </div>
      </div>
      <div className="flex gap-1.5">
        <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11 max-w-[100px]">
          <CodeBranch size="sm-regular" className="shrink-0 text-gray-12" />
          <span className="truncate">{deployment.gitBranch}</span>
        </div>
        <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded-md text-xs text-grayA-11">
          <CodeCommit size="sm-regular" className="shrink-0 text-gray-12" />
          <span>{shortenId(deployment.gitCommitSha ?? "")}</span>
        </div>
      </div>
    </div>
  </div>
);
