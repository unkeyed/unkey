"use client";

import { type Deployment, collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { CircleInfo, CodeBranch, CodeCommit } from "@unkey/icons";
import { Badge, Button, DialogContainer, toast } from "@unkey/ui";
import { StatusIndicator } from "../../details/active-deployment-card/status-indicator";

type DeploymentSectionProps = {
  title: string;
  deployment: Deployment;
  isActive: boolean;
  showSignal?: boolean;
};

const DeploymentSection = ({ title, deployment, isActive, showSignal }: DeploymentSectionProps) => (
  <div className="space-y-2">
    <div className="flex items-center gap-2">
      <h3 className="text-[13px] text-grayA-11">{title}</h3>
      <CircleInfo size="sm-regular" className="text-gray-9" />
    </div>
    <DeploymentCard deployment={deployment} isActive={isActive} showSignal={showSignal} />
  </div>
);

type RollbackDialogProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  deployment: Deployment;
  currentDeployment: Deployment;
  hostname?: string;
};

export const RollbackDialog = ({
  isOpen,
  onOpenChange,
  deployment,
  currentDeployment,
  hostname,
}: RollbackDialogProps) => {
  const utils = trpc.useUtils();
  const rollback = trpc.deploy.deployment.rollback.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Rollback completed", {
        description: `Successfully rolled back to deployment ${deployment.id}`,
      });
      // hack to revalidate
      try {
        // @ts-expect-error Their docs say it's here
        collection.projects.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }

      onOpenChange(false);
    },
    onError: (error) => {
      toast.error("Rollback failed", {
        description: error.message,
      });
    },
  });

  const handleRollback = async () => {
    if (!hostname) {
      toast.error("Missing hostname", {
        description: "Cannot perform rollback without hostname information",
      });
      return;
    }

    try {
      await rollback.mutateAsync({
        hostname,
        targetDeploymentId: deployment.id,
      });
    } catch (error) {
      console.error("Rollback error:", error);
    }
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title="Rollback to version"
      subTitle="Switch the active deployment to a target stable version"
      footer={
        <div className="flex flex-col items-center w-full gap-2">
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
          <div className="text-xs text-gray-9">Rollbacks usually complete within seconds</div>
        </div>
      }
    >
      <div className="space-y-9">
        <DeploymentSection
          title="Current active deployment"
          deployment={currentDeployment}
          isActive={true}
          showSignal={true}
        />
        <DeploymentSection title="Target version" deployment={deployment} isActive={false} />
      </div>
    </DialogContainer>
  );
};

type DeploymentCardProps = {
  deployment: Deployment;
  isActive: boolean;
  showSignal?: boolean;
};

const DeploymentCard = ({ deployment, isActive, showSignal }: DeploymentCardProps) => (
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
              variant={isActive ? "success" : "primary"}
              className={`px-1.5 ${isActive ? "text-successA-11" : "text-grayA-11"}`}
            >
              {isActive ? "Active" : "Preview"}
            </Badge>
          </div>
          <div className="text-xs text-grayA-9">
            {deployment.gitCommitMessage || `${isActive ? "Current active" : "Target"} deployment`}
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
