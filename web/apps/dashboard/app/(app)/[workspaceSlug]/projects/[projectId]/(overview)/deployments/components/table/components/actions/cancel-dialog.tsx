"use client";

import { type Deployment, collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentCard } from "./components/deployment-card";

type CancelDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  // Optional: fired once the cancel mutation succeeds. Lets the parent
  // hide the cancel-trigger button optimistically before the live query
  // observes the status change.
  onCancelled?: () => void;
  deployment: Deployment;
};

export const CancelDialog = ({ isOpen, onClose, onCancelled, deployment }: CancelDialogProps) => {
  const utils = trpc.useUtils();

  const cancel = trpc.deploy.deployment.cancel.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Deployment cancelled", {
        description: `Cancelled deployment ${deployment.id}`,
      });
      try {
        collection.deployments.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }
      onCancelled?.();
      onClose();
    },
    onError: (error) => {
      toast.error("Cancel failed", {
        description: error.message,
      });
    },
  });

  const handleCancel = async () => {
    await cancel
      .mutateAsync({
        deploymentId: deployment.id,
      })
      .catch((error) => {
        console.error("Cancel error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Cancel deployment"
      subTitle="This will permanently stop the current build. To deploy again, you'll need to trigger a new deployment."
      footer={
        <Button
          variant="destructive"
          size="xlg"
          onClick={handleCancel}
          disabled={cancel.isLoading}
          loading={cancel.isLoading}
          className="w-full rounded-lg"
        >
          Cancel deployment
        </Button>
      }
    >
      <DeploymentCard deployment={deployment} isCurrent={false} />
    </DialogContainer>
  );
};
