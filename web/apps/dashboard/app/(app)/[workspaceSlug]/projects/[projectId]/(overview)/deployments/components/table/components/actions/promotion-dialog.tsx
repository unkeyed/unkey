"use client";

import { type Deployment, collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentSection } from "./components/deployment-section";
import { DomainsSection } from "./components/domains-section";

type PromotionDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  targetDeployment: Deployment;
  currentDeployment: Deployment;
};

export const PromotionDialog = ({
  isOpen,
  onClose,
  targetDeployment,
  currentDeployment,
}: PromotionDialogProps) => {
  const utils = trpc.useUtils();
  const domains = useLiveQuery(
    (q) =>
      q
        .from({ domain: collection.domains })
        .where(({ domain }) => eq(domain.projectId, currentDeployment.projectId))
        .where(({ domain }) => inArray(domain.sticky, ["environment", "live"]))
        .where(({ domain }) => eq(domain.deploymentId, currentDeployment.id)),
    [currentDeployment.projectId, currentDeployment.id],
  );
  const promote = trpc.deploy.deployment.promote.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Promotion completed", {
        description: `Successfully promoted to deployment ${targetDeployment.id}`,
      });
      // hack to revalidate
      try {
        collection.projects.utils.refetch();
        collection.deployments.utils.refetch();
        collection.domains.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }

      onClose();
    },
    onError: (error) => {
      toast.error("Promotion failed", {
        description: error.message,
      });
    },
  });

  const handlePromotion = async () => {
    await promote
      .mutateAsync({
        targetDeploymentId: targetDeployment.id,
      })
      .catch((error) => {
        console.error("Promotion error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Promotion to version"
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
          Promote to
          {targetDeployment.gitCommitSha
            ? shortenId(targetDeployment.gitCommitSha)
            : targetDeployment.id}
        </Button>
      }
    >
      <div className="space-y-9">
        <DeploymentSection
          title="Current Deployment"
          deployment={currentDeployment}
          isCurrent={true}
          showSignal={true}
        />
        <DomainsSection domains={domains.data} />
        <DeploymentSection
          title="Target Deployment"
          deployment={targetDeployment}
          isCurrent={false}
        />
      </div>
    </DialogContainer>
  );
};
