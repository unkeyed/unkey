"use client";

import {
  type DeploymentSummary,
  toastWhenSettled,
  useDeployActions,
  useEnvironments,
} from "@/lib/deploy-data";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { Button, DialogContainer } from "@unkey/ui";
import { DeploymentCard } from "./components/deployment-card";

type StopDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  deployment: DeploymentSummary;
};

export const StopDialog = ({ isOpen, onClose, deployment }: StopDialogProps) => {
  const { awaitDeploymentStatus } = useDeployActions();
  const { data: environments = [] } = useEnvironments();
  const isProduction = environments.some(
    (environment) =>
      environment.id === deployment.environmentId && environment.kind === "production",
  );
  const stop = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.stopDeployment({ deploymentId }),
    onSuccess: onClose,
  });

  const handleStop = () => {
    toastWhenSettled(
      stop
        .mutateAsync(deployment.id)
        .then(() => awaitDeploymentStatus({ deploymentId: deployment.id, status: "stopped" })),
      {
        loading: "Stopping deployment...",
        success: { message: "Deployment stopped" },
        failure: (err) => ({
          message: "Failed to stop deployment",
          description: getErrorMessage(err),
        }),
      },
    );
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Stop deployment"
      subTitle={
        isProduction
          ? "Your current production deployment will keep running. Stopping this deployment removes instant rollback to it."
          : "Scale this deployment down. You can wake it again without creating a new deployment."
      }
      footer={
        <Button
          variant="destructive"
          size="md"
          onClick={handleStop}
          disabled={stop.isLoading}
          loading={stop.isLoading}
          className="w-full"
        >
          Stop deployment
        </Button>
      }
    >
      <DeploymentCard deployment={deployment} isCurrent={false} />
    </DialogContainer>
  );
};
