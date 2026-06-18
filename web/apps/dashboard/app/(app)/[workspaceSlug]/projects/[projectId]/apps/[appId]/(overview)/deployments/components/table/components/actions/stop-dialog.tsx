"use client";

import { type Deployment, collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentCard } from "./components/deployment-card";

type StopDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  deployment: Deployment;
};

export const StopDialog = ({ isOpen, onClose, deployment }: StopDialogProps) => {
  const utils = trpc.useUtils();

  const stop = trpc.deploy.deployment.stop.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Deployment stopping", {
        description: `Stopping deployment ${deployment.id}`,
      });
      try {
        collection.deployments.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }
      onClose();
    },
    onError: (error) => {
      toast.error("Stop failed", {
        description: error.message,
      });
    },
  });

  const handleStop = async () => {
    await stop
      .mutateAsync({
        deploymentId: deployment.id,
      })
      .catch((error) => {
        console.error("Stop error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Stop deployment"
      subTitle="Scale this deployment down. You can wake it again without creating a new deployment."
      footer={
        <Button
          variant="destructive"
          size="xlg"
          onClick={handleStop}
          disabled={stop.isLoading}
          loading={stop.isLoading}
          className="w-full rounded-lg"
        >
          Stop deployment
        </Button>
      }
    >
      <DeploymentCard deployment={deployment} isCurrent={false} />
    </DialogContainer>
  );
};
