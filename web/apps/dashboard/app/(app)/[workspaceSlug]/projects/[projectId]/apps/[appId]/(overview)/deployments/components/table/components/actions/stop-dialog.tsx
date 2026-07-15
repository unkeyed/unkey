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
  const stop = trpc.deploy.deployment.stop.useMutation({
    onSuccess: () => {
      collection.deployments.utils.refetch();
      onClose();
    },
  });

  const handleStop = () => {
    toast.promise(stop.mutateAsync({ deploymentId: deployment.id }), {
      loading: "Stopping deployment...",
      success: "Deployment stopped",
      error: (err) => ({
        message: "Failed to stop deployment",
        description: err.message,
      }),
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
