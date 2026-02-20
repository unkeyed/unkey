"use client";

import { type Deployment, collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentSection } from "./components/deployment-section";

type RedeployDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  selectedDeployment: Deployment;
};

export const RedeployDialog = ({ isOpen, onClose, selectedDeployment }: RedeployDialogProps) => {
  const utils = trpc.useUtils();

  const redeploy = trpc.deploy.deployment.redeploy.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Redeploy triggered", {
        description: "A new deployment has been queued",
      });
      try {
        collection.deployments.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }
      onClose();
    },
    onError: (error) => {
      toast.error("Redeploy failed", {
        description: error.message,
      });
    },
  });

  const handleRedeploy = async () => {
    await redeploy
      .mutateAsync({
        deploymentId: selectedDeployment.id,
      })
      .catch((error) => {
        console.error("Redeploy error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Redeploy"
      subTitle="Trigger a fresh build and deployment from the same branch"
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handleRedeploy}
          disabled={redeploy.isLoading}
          loading={redeploy.isLoading}
          className="w-full rounded-lg"
        >
          Redeploy
        </Button>
      }
    >
      <div className="space-y-9">
        <DeploymentSection title="Deployment" deployment={selectedDeployment} isLive={false} />
      </div>
    </DialogContainer>
  );
};
