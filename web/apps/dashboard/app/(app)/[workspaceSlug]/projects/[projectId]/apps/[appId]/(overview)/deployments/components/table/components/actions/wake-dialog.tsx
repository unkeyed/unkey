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
  const wake = trpc.deploy.deployment.wake.useMutation({
    onSuccess: () => {
      collection.deployments.utils.refetch();
      onClose();
    },
  });

  const handleWake = () => {
    toast.promise(wake.mutateAsync({ deploymentId: deployment.id }), {
      loading: "Waking deployment...",
      success: "Deployment is ready",
      error: (err) => ({
        message: "Failed to wake deployment",
        description: err.message,
      }),
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
