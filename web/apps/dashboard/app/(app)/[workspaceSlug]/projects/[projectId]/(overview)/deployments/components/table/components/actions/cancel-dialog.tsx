"use client";

import { type Deployment, collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentSection } from "./components/deployment-section";

type CancelDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  selectedDeployment: Deployment;
};

export const CancelDialog = ({ isOpen, onClose, selectedDeployment }: CancelDialogProps) => {
  const utils = trpc.useUtils();

  const cancelDeployment = trpc.deploy.deployment.cancel.useMutation({
    onSuccess: () => {
      utils.invalidate();
      try {
        collection.deployments.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }
      onClose();
      toast.success("Deployment cancelled");
    },
    onError: (error) => {
      toast.error("Failed to cancel deployment", {
        description: error.message,
      });
    },
  });

  const handleCancel = async () => {
    await cancelDeployment
      .mutateAsync({
        deploymentId: selectedDeployment.id,
      })
      .catch((error) => {
        console.error("Cancel error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Cancel Deployment"
      subTitle="This will stop the deployment process. This action cannot be undone."
      footer={
        <Button
          variant="destructive"
          size="xlg"
          onClick={handleCancel}
          disabled={cancelDeployment.isLoading}
          loading={cancelDeployment.isLoading}
          className="w-full rounded-lg"
        >
          Cancel Deployment
        </Button>
      }
    >
      <div className="space-y-9">
        <DeploymentSection title="Deployment" deployment={selectedDeployment} isCurrent={false} />
      </div>
    </DialogContainer>
  );
};
