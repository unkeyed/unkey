"use client";

import { type DeploymentSummary, toastWhenSettled, useDeployActions } from "@/lib/deploy-data";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { Button, DialogContainer } from "@unkey/ui";
import { DeploymentCard } from "./components/deployment-card";

type WakeDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  deployment: DeploymentSummary;
};

export const WakeDialog = ({ isOpen, onClose, deployment }: WakeDialogProps) => {
  const { awaitDeploymentStatus } = useDeployActions();
  const wake = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.startDeployment({ deploymentId }),
    onSuccess: onClose,
  });

  const handleWake = () => {
    toastWhenSettled(
      wake
        .mutateAsync(deployment.id)
        .then(() => awaitDeploymentStatus({ deploymentId: deployment.id, status: "ready" })),
      {
        loading: "Waking deployment...",
        success: { message: "Deployment is ready" },
        failure: (err) => ({
          message: "Failed to wake deployment",
          description: getErrorMessage(err),
        }),
      },
    );
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
          size="md"
          onClick={handleWake}
          disabled={wake.isLoading}
          loading={wake.isLoading}
          className="w-full"
        >
          Wake deployment
        </Button>
      }
    >
      <DeploymentCard deployment={deployment} isCurrent={false} />
    </DialogContainer>
  );
};
