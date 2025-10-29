"use client";

import { type Deployment, collection, collectionManager } from "@/lib/collections";
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
  liveDeployment: Deployment;
};

export const PromotionDialog = ({
  isOpen,
  onClose,
  targetDeployment,
  liveDeployment,
}: PromotionDialogProps) => {
  const utils = trpc.useUtils();
  const domainCollection = collectionManager.getProjectCollections(
    liveDeployment.projectId,
  ).domains;
  const domains = useLiveQuery((q) =>
    q
      .from({ domain: domainCollection })
      .where(({ domain }) => inArray(domain.sticky, ["environment", "live"]))
      .where(({ domain }) => eq(domain.deploymentId, liveDeployment.id)),
  );
  const promote = trpc.deploy.deployment.promote.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Promotion completed", {
        description: `Successfully promoted to deployment ${targetDeployment.id}`,
      });
      // hack to revalidate
      try {
        // @ts-expect-error Their docs say it's here
        collection.projects.utils.refetch();
        // @ts-expect-error Their docs say it's here
        collection.deployments.utils.refetch();
        // @ts-expect-error Their docs say it's here
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
          title="Live Deployment"
          deployment={liveDeployment}
          isLive={true}
          showSignal={true}
        />
        <DomainsSection domains={domains.data} />
        <DeploymentSection title="Target Deployment" deployment={targetDeployment} isLive={false} />
      </div>
    </DialogContainer>
  );
};
