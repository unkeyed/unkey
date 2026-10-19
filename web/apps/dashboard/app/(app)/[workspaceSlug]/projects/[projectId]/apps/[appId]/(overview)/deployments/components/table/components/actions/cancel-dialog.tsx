"use client";

import type { DeploymentSummary } from "@/lib/collections";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useProjectData } from "../../../../../data-provider";
import { DeploymentCard } from "./components/deployment-card";

type CancelDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  // Optional: fired once the cancel mutation succeeds. Lets the parent
  // hide the cancel-trigger button optimistically before the live query
  // observes the status change.
  onCancelled?: () => void;
  deployment: DeploymentSummary;
};

export const CancelDialog = ({ isOpen, onClose, onCancelled, deployment }: CancelDialogProps) => {
  const { awaitDeploymentStatus } = useProjectData();

  const cancel = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.cancelDeployment({ deploymentId }),
    onSuccess: () => {
      awaitDeploymentStatus({ deploymentId: deployment.id, status: "cancelled" });
      onCancelled?.();
      onClose();
    },
  });

  const handleCancel = () => {
    toast.promise(cancel.mutateAsync(deployment.id), {
      loading: "Cancelling deployment...",
      success: "Deployment cancelled",
      error: (err) => getErrorToast(err, "Failed to cancel deployment"),
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
