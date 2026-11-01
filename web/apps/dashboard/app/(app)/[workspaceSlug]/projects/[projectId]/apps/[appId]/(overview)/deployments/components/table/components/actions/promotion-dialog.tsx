"use client";

import { type DeploymentSummary, useDeployActions, useDomains } from "@/lib/deploy-data";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentSection } from "./components/deployment-section";
import { DomainsSection } from "./components/domains-section";

type PromotionDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  targetDeployment: DeploymentSummary;
  currentDeployment: DeploymentSummary;
};

export const PromotionDialog = ({
  isOpen,
  onClose,
  targetDeployment,
  currentDeployment,
}: PromotionDialogProps) => {
  const { awaitLiveDeployment } = useDeployActions();
  const { data: appDomains } = useDomains();
  const domains = appDomains.filter(
    (d) =>
      d.deploymentId === currentDeployment.id &&
      (d.sticky === "environment" || d.sticky === "live"),
  );
  const promote = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.promoteDeployment({ deploymentId }),
    onSuccess: () => {
      awaitLiveDeployment({ deploymentId: targetDeployment.id, rolledBack: false });
      toast.success("Promotion completed", {
        description: `Successfully promoted to deployment ${targetDeployment.id}`,
      });
      onClose();
    },
    onError: (error) => {
      toast.error("Promotion failed", {
        description: getErrorMessage(error),
      });
    },
  });

  const handlePromotion = async () => {
    await promote.mutateAsync(targetDeployment.id).catch((error) => {
      console.error("Promotion error:", error);
    });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Promote to version"
      subTitle="Switch the active deployment to a target stable version"
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handlePromotion}
          disabled={promote.isLoading}
          loading={promote.isLoading}
          className="w-full rounded-lg"
        >
          Promote target deployment
        </Button>
      }
    >
      <div className="flex flex-col gap-9">
        <DeploymentSection
          title="Current Deployment"
          deployment={currentDeployment}
          isCurrent={true}
        />
        <DomainsSection domains={domains} />
        <DeploymentSection
          title="Target Deployment"
          deployment={targetDeployment}
          isCurrent={false}
        />
      </div>
    </DialogContainer>
  );
};
