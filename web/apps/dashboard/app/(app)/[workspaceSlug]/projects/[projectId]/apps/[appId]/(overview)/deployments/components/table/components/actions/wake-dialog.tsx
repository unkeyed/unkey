"use client";

import { useMutation } from "@tanstack/react-query";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { collection, type Deployment } from "@/lib/collections";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { DeploymentCard } from "./components/deployment-card";

type WakeDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  deployment: Deployment;
};

export const WakeDialog = ({ isOpen, onClose, deployment }: WakeDialogProps) => {
  const wake = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.startDeployment({ deploymentId }),
    onSuccess: () => {
      collection.deployments.utils.refetch();
      onClose();
    },
  });

  const handleWake = () => {
    toast.promise(wake.mutateAsync(deployment.id), {
      loading: "Waking deployment...",
      success: "Deployment is ready",
      error: (err) => ({
        message: "Failed to wake deployment",
        description: getErrorMessage(err),
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
