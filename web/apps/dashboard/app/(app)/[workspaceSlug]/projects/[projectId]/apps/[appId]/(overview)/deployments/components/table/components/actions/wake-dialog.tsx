"use client";

import { type Deployment, collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentCard } from "./components/deployment-card";

type WakeDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  deployment: Deployment;
};

export const WakeDialog = ({ isOpen, onClose, deployment }: WakeDialogProps) => {
  const utils = trpc.useUtils();

  const wake = trpc.deploy.deployment.wake.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Deployment waking", {
        description: `Waking deployment ${deployment.id}`,
      });
      try {
        collection.deployments.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }
      onClose();
    },
    onError: (error) => {
      toast.error("Wake failed", {
        description: error.message,
      });
    },
  });

  const handleWake = async () => {
    await wake
      .mutateAsync({
        deploymentId: deployment.id,
      })
      .catch((error) => {
        console.error("Wake error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Wake deployment"
      subTitle="Scale this stopped deployment back up and wait until it is ready."
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handleWake}
          disabled={wake.isLoading}
          loading={wake.isLoading}
          className="w-full rounded-lg"
        >
          Wake deployment
        </Button>
      }
    >
      <DeploymentCard deployment={deployment} isCurrent={false} />
    </DialogContainer>
  );
};
